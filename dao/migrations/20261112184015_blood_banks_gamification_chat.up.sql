-- =====================================================================
-- 000003_blood_banks_gamification_chat (UP)
--
--   1. users.upazila / points / donation_count, donation_posts.urgency
--   2. Blood banks (upazila-level donor groups) and their members
--   3. Gamification: point ledger, badges, user badges and backfill of
--      points / badges from the existing donation_history
--   4. Realtime chat: conversations, participants, messages
--   5. MANAGE_BLOOD_BANKS permission (id 10, see rest/permissionList.go)
--
-- Reverse with 000003_blood_banks_gamification_chat.down.sql.
-- =====================================================================


-- =====================================================================
-- 1. USERS / POSTS
-- =====================================================================
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS upazila        varchar(80);
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS points         integer NOT NULL DEFAULT 0;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS donation_count integer NOT NULL DEFAULT 0;

ALTER TABLE public.donation_posts ADD COLUMN IF NOT EXISTS urgency varchar(20) NOT NULL DEFAULT 'normal'
    CHECK (urgency IN ('normal','urgent','emergency'));


-- =====================================================================
-- 2. BLOOD BANKS
-- =====================================================================
CREATE TABLE IF NOT EXISTS public.blood_banks (
    id               serial PRIMARY KEY,
    name             varchar(120) NOT NULL,
    description      text,
    division         varchar(100) NOT NULL,
    district         varchar(100) NOT NULL,
    upazila          varchar(80)  NOT NULL,
    address          text,
    contact_phone    varchar(50),
    logo             varchar(255),
    status           varchar(20)  NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','approved','rejected','suspended')),
    admin_id         integer NOT NULL REFERENCES public.users(id),
    created_by       integer NOT NULL REFERENCES public.users(id),
    reviewed_by      integer REFERENCES public.users(id) ON DELETE SET NULL,
    reviewed_at      timestamptz,
    rejection_reason varchar(500),
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_blood_banks_location ON public.blood_banks (division, district, upazila, status);
CREATE INDEX IF NOT EXISTS idx_blood_banks_status   ON public.blood_banks (status);
-- a donor can have only one creation request waiting for review
CREATE UNIQUE INDEX IF NOT EXISTS ux_blood_banks_one_pending_per_creator
    ON public.blood_banks (created_by) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS public.blood_bank_members (
    id            serial PRIMARY KEY,
    blood_bank_id integer NOT NULL REFERENCES public.blood_banks(id) ON DELETE CASCADE,
    user_id       integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    role          varchar(20) NOT NULL DEFAULT 'member' CHECK (role IN ('admin','officer','member')),
    position      varchar(60),
    status        varchar(20) NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','active','rejected','left','removed')),
    approved_by   integer REFERENCES public.users(id) ON DELETE SET NULL,
    joined_at     timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_blood_bank_members_bank ON public.blood_bank_members (blood_bank_id, status);
-- a donor belongs to (or is waiting to join) at most one blood bank
CREATE UNIQUE INDEX IF NOT EXISTS ux_blood_bank_members_one_bank
    ON public.blood_bank_members (user_id) WHERE status IN ('pending','active');


-- =====================================================================
-- 3. GAMIFICATION
-- =====================================================================
ALTER TABLE public.donation_history ADD COLUMN IF NOT EXISTS blood_bank_id  integer REFERENCES public.blood_banks(id) ON DELETE SET NULL;
ALTER TABLE public.donation_history ADD COLUMN IF NOT EXISTS confirmed_by   integer REFERENCES public.users(id) ON DELETE SET NULL;
ALTER TABLE public.donation_history ADD COLUMN IF NOT EXISTS points_awarded integer NOT NULL DEFAULT 0;
-- a requester can confirm a donor only once per post
CREATE UNIQUE INDEX IF NOT EXISTS ux_donation_history_post_donor
    ON public.donation_history (post_id, donor_id) WHERE post_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_donation_history_bank ON public.donation_history (blood_bank_id);

-- Ledger of every point change; users.points is the cached sum.
CREATE TABLE IF NOT EXISTS public.point_transactions (
    id         serial PRIMARY KEY,
    user_id    integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    points     integer NOT NULL,
    reason     varchar(30) NOT NULL
               CHECK (reason IN ('donation','first_donation','emergency_bonus','bank_founder')),
    ref_type   varchar(30),
    ref_id     integer,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_point_transactions_user ON public.point_transactions (user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_point_transactions_date ON public.point_transactions (created_at);

CREATE TABLE IF NOT EXISTS public.badges (
    id          serial PRIMARY KEY,
    code        varchar(40) NOT NULL UNIQUE,
    name        varchar(80) NOT NULL,
    description varchar(255) NOT NULL,
    icon        varchar(40) NOT NULL,
    kind        varchar(20) NOT NULL CHECK (kind IN ('milestone','periodic','special')),
    threshold   integer NOT NULL DEFAULT 0,
    sort_order  integer NOT NULL DEFAULT 0
);

INSERT INTO public.badges (code, name, description, icon, kind, threshold, sort_order) VALUES
('first_drop',          'First Drop',          'Made your first confirmed donation',            'droplet',  'milestone', 1,  1),
('life_saver',          'Life Saver',          '3 confirmed donations',                          'heart',    'milestone', 3,  2),
('hero',                'Hero',                '5 confirmed donations',                          'shield',   'milestone', 5,  3),
('super_hero',          'Super Hero',          '10 confirmed donations',                         'zap',      'milestone', 10, 4),
('legend',              'Legend',              '25 confirmed donations',                         'crown',    'milestone', 25, 5),
('emergency_responder', 'Emergency Responder', 'Donated for an emergency request',               'siren',    'special',   0,  6),
('bank_founder',        'Bank Founder',        'Founded a blood bank that was approved',         'landmark', 'special',   0,  7),
('top_donor_month',     'Top Donor of the Month', 'Most points in your blood bank for a month',  'trophy',   'periodic',  0,  8)
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS public.user_badges (
    id            serial PRIMARY KEY,
    user_id       integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    badge_id      integer NOT NULL REFERENCES public.badges(id) ON DELETE CASCADE,
    blood_bank_id integer REFERENCES public.blood_banks(id) ON DELETE SET NULL,
    period        varchar(7) NOT NULL DEFAULT '',   -- 'YYYY-MM' for periodic badges
    awarded_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, badge_id, period)
);
CREATE INDEX IF NOT EXISTS idx_user_badges_user ON public.user_badges (user_id);

-- Backfill from the existing donation history so leaderboards are not empty.
INSERT INTO public.point_transactions (user_id, points, reason, ref_type, ref_id, created_at)
SELECT h.donor_id, 100, 'donation', 'donation_history', h.id, h.donation_date::timestamptz
FROM public.donation_history h
WHERE NOT EXISTS (SELECT 1 FROM public.point_transactions t
                  WHERE t.ref_type = 'donation_history' AND t.ref_id = h.id AND t.reason = 'donation');

INSERT INTO public.point_transactions (user_id, points, reason, ref_type, ref_id, created_at)
SELECT DISTINCT ON (h.donor_id) h.donor_id, 50, 'first_donation', 'donation_history', h.id, h.donation_date::timestamptz
FROM public.donation_history h
WHERE NOT EXISTS (SELECT 1 FROM public.point_transactions t
                  WHERE t.user_id = h.donor_id AND t.reason = 'first_donation')
ORDER BY h.donor_id, h.donation_date, h.id;

UPDATE public.donation_history h SET points_awarded = (
    SELECT COALESCE(SUM(t.points), 0) FROM public.point_transactions t
    WHERE t.ref_type = 'donation_history' AND t.ref_id = h.id);

UPDATE public.users u SET
    points         = COALESCE((SELECT SUM(t.points) FROM public.point_transactions t WHERE t.user_id = u.id), 0),
    donation_count = (SELECT COUNT(*) FROM public.donation_history h WHERE h.donor_id = u.id);

INSERT INTO public.user_badges (user_id, badge_id)
SELECT u.id, b.id
FROM public.users u
JOIN public.badges b ON b.kind = 'milestone' AND u.donation_count >= b.threshold
ON CONFLICT (user_id, badge_id, period) DO NOTHING;


-- =====================================================================
-- 4. CHAT
-- =====================================================================
CREATE TABLE IF NOT EXISTS public.conversations (
    id               serial PRIMARY KEY,
    type             varchar(10) NOT NULL CHECK (type IN ('bank','direct')),
    blood_bank_id    integer UNIQUE REFERENCES public.blood_banks(id) ON DELETE CASCADE,
    post_id          integer REFERENCES public.donation_posts(id) ON DELETE SET NULL,
    -- the two users of a direct chat, lower id first
    direct_user_low  integer REFERENCES public.users(id) ON DELETE CASCADE,
    direct_user_high integer REFERENCES public.users(id) ON DELETE CASCADE,
    created_at       timestamptz NOT NULL DEFAULT now(),
    last_message_at  timestamptz,
    CHECK ((type = 'bank'   AND blood_bank_id IS NOT NULL)
        OR (type = 'direct' AND direct_user_low IS NOT NULL AND direct_user_high IS NOT NULL
                            AND direct_user_low < direct_user_high))
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_conversations_direct
    ON public.conversations (post_id, direct_user_low, direct_user_high) WHERE type = 'direct';

CREATE TABLE IF NOT EXISTS public.conversation_participants (
    conversation_id      integer NOT NULL REFERENCES public.conversations(id) ON DELETE CASCADE,
    user_id              integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    last_read_message_id bigint NOT NULL DEFAULT 0,
    joined_at            timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_conversation_participants_user ON public.conversation_participants (user_id);

CREATE TABLE IF NOT EXISTS public.messages (
    id              bigserial PRIMARY KEY,
    conversation_id integer NOT NULL REFERENCES public.conversations(id) ON DELETE CASCADE,
    sender_id       integer REFERENCES public.users(id) ON DELETE SET NULL,
    body            text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON public.messages (conversation_id, id DESC);


-- =====================================================================
-- 5. PERMISSION (id MUST match rest/permissionList.go)
-- =====================================================================
INSERT INTO public.permission (id, name, only_superadmin, description) VALUES
(10, 'MANAGE_BLOOD_BANKS', false, 'Review, approve, reject and suspend blood banks')
ON CONFLICT (id) DO NOTHING;

SELECT setval('public.permission_id_seq', (SELECT MAX(id) FROM public.permission));

INSERT INTO public.role_permission (role_id, permission_id)
SELECT v.role_id, v.permission_id
FROM (VALUES (1, 10), (2, 10)) AS v(role_id, permission_id)
WHERE NOT EXISTS (
    SELECT 1 FROM public.role_permission rp
    WHERE rp.role_id = v.role_id AND rp.permission_id = v.permission_id
);
