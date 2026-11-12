-- =====================================================================
-- 000003_blood_banks_gamification_chat (DOWN)
-- Removes blood banks, gamification and chat. WARNING: this deletes all
-- blood banks, chat messages, points and badges.
-- =====================================================================

DELETE FROM public.role_permission WHERE permission_id = 10;
DELETE FROM public.permission WHERE id = 10 AND name = 'MANAGE_BLOOD_BANKS';
SELECT setval('public.permission_id_seq',
              COALESCE((SELECT MAX(id) FROM public.permission), 1),
              (SELECT MAX(id) FROM public.permission) IS NOT NULL);

DROP TABLE IF EXISTS public.messages;
DROP TABLE IF EXISTS public.conversation_participants;
DROP TABLE IF EXISTS public.conversations;

DROP TABLE IF EXISTS public.user_badges;
DROP TABLE IF EXISTS public.badges;
DROP TABLE IF EXISTS public.point_transactions;

DROP INDEX IF EXISTS public.idx_donation_history_bank;
DROP INDEX IF EXISTS public.ux_donation_history_post_donor;
ALTER TABLE public.donation_history DROP COLUMN IF EXISTS points_awarded;
ALTER TABLE public.donation_history DROP COLUMN IF EXISTS confirmed_by;
ALTER TABLE public.donation_history DROP COLUMN IF EXISTS blood_bank_id;

DROP TABLE IF EXISTS public.blood_bank_members;
DROP TABLE IF EXISTS public.blood_banks;

ALTER TABLE public.donation_posts DROP COLUMN IF EXISTS urgency;

ALTER TABLE public.users DROP COLUMN IF EXISTS donation_count;
ALTER TABLE public.users DROP COLUMN IF EXISTS points;
ALTER TABLE public.users DROP COLUMN IF EXISTS upazila;
