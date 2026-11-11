-- =====================================================================
-- 000002_seed_dummy_data (UP)
-- Realistic demo data for BloodLink (Bangladesh blood-donor network).
--
--   1. Schema the site needs but 000001 lacks:
--        users.blood_group / users.date_of_birth / users.photo
--        donation_requests, donation_posts, donation_post_interests,
--        donation_history
--   2. Permissions + role_permission
--   3. 61 users (1 test account + 3 managers + 57 donors) across all
--      8 divisions
--   4. user_locations, sessions (expired only)
--   5. Donation requests, posts, interests and donation history that
--      are consistent with each other and with users.donated_at
--   6. Blood-request test scenarios:
--        donor@blood-link.online has incoming AND outgoing requests in
--        every status (pending / accepted / rejected / completed);
--        superadmin and admin have open emergency requests; plus an
--        overdue, a soft-deleted, a 6-unit, an unverified-requester,
--        a month-ahead and a hidden-donor request.
--
-- Runs after 000001, so it uses the normalized snake_case column names.
-- Every insert is idempotent (ON CONFLICT / NOT EXISTS).
-- Reverse with 000002_seed_dummy_data.down.sql.
--
-- All seeded users share the password: Admin@1234
-- Dates are relative to the migration time so the data looks current.
-- =====================================================================


-- =====================================================================
-- 1. SCHEMA
-- =====================================================================
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS blood_group   varchar(5);
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS date_of_birth date;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS photo         varchar(255);

-- Blood requests: sent to a specific donor (donor_id set) or open to all.
CREATE TABLE IF NOT EXISTS public.donation_requests (
    id               serial PRIMARY KEY,
    requester_id     integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    donor_id         integer REFERENCES public.users(id) ON DELETE SET NULL,
    blood_group      varchar(5)   NOT NULL CHECK (blood_group IN ('A+','A-','B+','B-','AB+','AB-','O+','O-')),
    urgency          varchar(20)  NOT NULL DEFAULT 'normal' CHECK (urgency IN ('normal','urgent','emergency')),
    patient_name     varchar(255) NOT NULL,
    reason           varchar(255),
    units_needed     integer      NOT NULL DEFAULT 1 CHECK (units_needed > 0),
    hospital_name    varchar(255) NOT NULL,
    address          text,
    district         varchar(100),
    division         varchar(100),
    contact_phone    varchar(50),
    required_date    date         NOT NULL,
    status           varchar(20)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected','completed')),
    is_admin_request boolean      NOT NULL DEFAULT false,
    is_deleted       boolean      NOT NULL DEFAULT false,
    created_at       timestamp    DEFAULT now(),
    updated_at       timestamp    DEFAULT now(),
    created_by       integer,
    updated_by       integer
);
CREATE INDEX IF NOT EXISTS idx_donation_requests_status    ON public.donation_requests (status);
CREATE INDEX IF NOT EXISTS idx_donation_requests_bg_div    ON public.donation_requests (blood_group, division);
CREATE INDEX IF NOT EXISTS idx_donation_requests_donor     ON public.donation_requests (donor_id);
CREATE INDEX IF NOT EXISTS idx_donation_requests_requester ON public.donation_requests (requester_id);

-- Public "blood needed" posts.
CREATE TABLE IF NOT EXISTS public.donation_posts (
    id            serial PRIMARY KEY,
    author_id     integer      NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    blood_group   varchar(5)   NOT NULL CHECK (blood_group IN ('A+','A-','B+','B-','AB+','AB-','O+','O-')),
    title         varchar(255) NOT NULL,
    description   text,
    hospital_name varchar(255),
    address       text,
    district      varchar(100),
    division      varchar(100),
    units_needed  integer      NOT NULL DEFAULT 1 CHECK (units_needed > 0),
    contact_phone varchar(50),
    required_date date,
    status        varchar(20)  NOT NULL DEFAULT 'open' CHECK (status IN ('open','fulfilled','closed')),
    is_deleted    boolean      NOT NULL DEFAULT false,
    created_at    timestamp    DEFAULT now(),
    updated_at    timestamp    DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_donation_posts_status ON public.donation_posts (status);
CREATE INDEX IF NOT EXISTS idx_donation_posts_bg_div ON public.donation_posts (blood_group, division);

-- Users who tapped "I'm interested" on a post.
CREATE TABLE IF NOT EXISTS public.donation_post_interests (
    id         serial PRIMARY KEY,
    post_id    integer NOT NULL REFERENCES public.donation_posts(id) ON DELETE CASCADE,
    user_id    integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    created_at timestamp DEFAULT now(),
    UNIQUE (post_id, user_id)
);

-- Completed donations.
CREATE TABLE IF NOT EXISTS public.donation_history (
    id             serial PRIMARY KEY,
    donor_id       integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    recipient_id   integer REFERENCES public.users(id) ON DELETE SET NULL,
    recipient_name varchar(255),
    request_id     integer REFERENCES public.donation_requests(id) ON DELETE SET NULL,
    post_id        integer REFERENCES public.donation_posts(id) ON DELETE SET NULL,
    blood_group    varchar(5) NOT NULL CHECK (blood_group IN ('A+','A-','B+','B-','AB+','AB-','O+','O-')),
    units          integer    NOT NULL DEFAULT 1 CHECK (units > 0),
    hospital_name  varchar(255),
    district       varchar(100),
    division       varchar(100),
    donation_date  date       NOT NULL,
    created_at     timestamp  DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_donation_history_donor ON public.donation_history (donor_id);
CREATE INDEX IF NOT EXISTS idx_donation_history_date  ON public.donation_history (donation_date);


-- =====================================================================
-- 2. DEFAULT ADMINS: fill the new profile fields
-- =====================================================================
UPDATE public.users
SET blood_group = 'O+', country = 'Bangladesh', city = 'Dhaka',
    district = 'Dhaka', division = 'Dhaka'
WHERE email IN ('superadmin@blood-link.online', 'admin@blood-link.online')
  AND blood_group IS NULL;


-- =====================================================================
-- 3. PERMISSIONS (ids MUST match rest/permissionList.go iota order)
-- =====================================================================
INSERT INTO public.permission (id, name, only_superadmin, description) VALUES
(1, 'CHANGE_PASSWORD',    false, 'Change own account password'),
(2, 'RESET_PASSWORD',     true,  'Reset another user''s password and issue a temporary one'),
(3, 'CREATE_USER',        false, 'Create new donor, manager or admin accounts'),
(4, 'UPDATE_USER',        false, 'Edit user profile information'),
(5, 'LIST_USER',          false, 'Browse and search the donor directory'),
(6, 'VIEW_USER',          false, 'View a single donor profile'),
(7, 'DELETE_USER',        true,  'Permanently remove a user account'),
(8, 'TOGGLE_USER_STATUS', false, 'Enable or disable a user account'),
(9, 'PROFILE_LOGIN',      false, 'Log in to a user profile')
ON CONFLICT (id) DO NOTHING;

SELECT setval('public.permission_id_seq', (SELECT MAX(id) FROM public.permission));

-- 1 SUPERADMIN: everything | 2 ADMIN: all but superadmin-only
-- 3 MANAGER: donor management | 4 USERROLE: regular donor
INSERT INTO public.role_permission (role_id, permission_id)
SELECT v.role_id, v.permission_id
FROM (VALUES
    (1, 1), (1, 2), (1, 3), (1, 4), (1, 5), (1, 6), (1, 7), (1, 8), (1, 9),
    (2, 1), (2, 3), (2, 4), (2, 5), (2, 6), (2, 8), (2, 9),
    (3, 1), (3, 4), (3, 5), (3, 6), (3, 8), (3, 9),
    (4, 1), (4, 4), (4, 5), (4, 6), (4, 9)
) AS v(role_id, permission_id)
WHERE NOT EXISTS (
    SELECT 1 FROM public.role_permission rp
    WHERE rp.role_id = v.role_id AND rp.permission_id = v.permission_id
);


-- =====================================================================
-- 4. USERS
--    Staged in a temp table so later steps touch ONLY seeded users.
--    donated / login / created = days ago (NULL = never)
-- =====================================================================
DROP TABLE IF EXISTS pg_temp.seed_users;
CREATE TEMP TABLE seed_users AS
SELECT * FROM (VALUES
-- ===== Test account: has requests in every status (see section 7) =====
('Demo',        'Donor',     'M', 'donor@blood-link.online',           '+8801700112233', 'House 10, Road 2, Dhanmondi',              'Dhaka',              'Dhaka',        'Dhaka',      'B+',  'QA Engineer',              4, true,  false, 150,  0,    200, 1),
-- ===== Managers =====
('Kamrul',     'Hasan',     'M', 'kamrul.hasan@blood-link.online',    '+8801711458236', 'House 14, Road 3, Block B, Mirpur 10',     'Dhaka',              'Dhaka',        'Dhaka',      'B+',  'Volunteer Coordinator',    3, true,  false, 140,  1,    520, 1),
('Shamima',     'Nasrin',    'F', 'shamima.nasrin@blood-link.online',  '+8801819264573', 'Flat 5B, 22/A Chatteshwari Road',          'Chattogram',         'Chattogram',   'Chattogram', 'O+',  'Community Health Officer', 3, true,  false, 210,  2,    480, 1),
('Abu Sayeed',  'Chowdhury', 'M', 'abu.sayeed@blood-link.online',      '+8801912384756', 'House 7, Lamabazar',                       'Sylhet',             'Sylhet',       'Sylhet',     'A+',  'Program Manager',          3, true,  false, 95,   3,    450, 2),
-- ===== Dhaka division =====
('Tanvir',      'Ahmed',     'M', 'tanvir.ahmed92@gmail.com',          '+8801712345981', 'House 23, Road 7, Dhanmondi',              'Dhaka',              'Dhaka',        'Dhaka',      'B+',  'Software Engineer',        4, true,  false, 128,  1,    410, NULL),
('Nusrat',      'Jahan',     'F', 'nusrat.jahan@yahoo.com',            '+8801819456732', 'Flat 4A, 12 Kemal Ataturk Avenue, Banani', 'Dhaka',              'Dhaka',        'Dhaka',      'O+',  'Doctor',                   4, true,  true,  NULL, 3,    220, NULL),
('Mahmudul',    'Hasan',     'M', 'mahmud.hasan.du@gmail.com',         '+8801674239815', 'Shahidullah Hall, University of Dhaka',    'Dhaka',              'Dhaka',        'Dhaka',      'A+',  'Student',                  4, true,  false, 45,   0,    300, NULL),
('Sadia',       'Islam',     'F', 'sadia.islam07@gmail.com',           '+8801534872910', 'House 9, Road 11, Sector 4, Uttara',       'Dhaka',              'Dhaka',        'Dhaka',      'B+',  'Banker',                   4, true,  false, 190,  5,    380, NULL),
('Rafiqul',     'Islam',     'M', 'rafiqul.islam.savar@gmail.com',     '+8801911227364', 'Bank Colony, Savar',                       'Savar',              'Dhaka',        'Dhaka',      'O+',  'Garments Merchandiser',    4, true,  false, 260,  12,   350, 2),
('Farhana',     'Akter',     'F', 'farhana.akter88@gmail.com',         '+8801748561239', 'Chandra, Kaliakair',                       'Kaliakair',          'Gazipur',      'Dhaka',      'AB+', 'Teacher',                  4, true,  false, 330,  9,    400, NULL),
('Arif',        'Hossain',   'M', 'arif.hossain.bd@outlook.com',       '+8801556783412', 'Board Bazar, Gazipur Sadar',               'Gazipur Sadar',      'Gazipur',      'Dhaka',      'B+',  'Electrical Engineer',      4, true,  false, 75,   2,    290, NULL),
('Sumaiya',     'Rahman',    'F', 'sumaiya.rahman@gmail.com',          '+8801782654310', '34/2 Green Road, Panthapath',              'Dhaka',              'Dhaka',        'Dhaka',      'A-',  'Pharmacist',               4, true,  true,  160,  4,    260, NULL),
('Shakil',      'Ahmed',     'M', 'shakil.ahmed.ng@gmail.com',         '+8801816347290', 'Chashara, Narayanganj Sadar',              'Narayanganj Sadar',  'Narayanganj',  'Dhaka',      'O+',  'Businessman',              4, true,  false, NULL, 20,   180, NULL),
('Ayesha',      'Siddika',   'F', 'ayesha.siddika@gmail.com',          '+8801923478156', 'Shantinagar, Tangail Sadar',               'Tangail Sadar',      'Tangail',      'Dhaka',      'B-',  'Nurse',                    4, true,  false, 230,  6,    340, 1),
('Imran',       'Kabir',     'M', 'imran.kabir.mpur@gmail.com',        '+8801713590284', 'House 18, Block D, Mohammadpur',           'Dhaka',              'Dhaka',        'Dhaka',      'O-',  'Govt. Officer',            4, true,  false, 112,  1,    470, NULL),
('Jannatul',    'Ferdous',   'F', 'jannatul.ferdous99@gmail.com',      '+8801687453201', 'Road 5, Mirpur DOHS',                      'Dhaka',              'Dhaka',        'Dhaka',      'A+',  'Student',                  4, false, false, NULL, NULL, 25,   NULL),
('Sabbir',      'Rahman',    'M', 'sabbir.rahman@gmail.com',           '+8801974512368', 'Road 8, Block C, Bashundhara R/A',         'Dhaka',              'Dhaka',        'Dhaka',      'B+',  'Marketing Executive',      4, true,  false, 98,   2,    330, NULL),
('Tahmina',     'Akhter',    'F', 'tahmina.akhter@yahoo.com',          '+8801552348761', 'Sreenagar Bazar',                          'Sreenagar',          'Munshiganj',   'Dhaka',      'O+',  'Homemaker',                4, true,  true,  NULL, 40,   210, NULL),
('Nazmul',      'Huda',      'M', 'nazmul.huda.fp@gmail.com',          '+8801716783429', 'Jhiltuli, Faridpur Sadar',                 'Faridpur Sadar',     'Faridpur',     'Dhaka',      'AB+', 'Lawyer',                   4, true,  false, 205,  7,    360, NULL),
('Mehedi',      'Hasan',     'M', 'mehedi.hasan.bhairab@gmail.com',    '+8801825641937', 'Bhairab Bazar',                            'Bhairab',            'Kishoreganj',  'Dhaka',      'B+',  'Shop Owner',               4, true,  false, 67,   3,    150, NULL),
('Rumana',      'Afroz',     'F', 'rumana.afroz@gmail.com',            '+8801632894175', 'Road 21, Mohakhali DOHS',                  'Dhaka',              'Dhaka',        'Dhaka',      'A+',  'Architect',                4, true,  false, 300,  4,    420, NULL),
('Asif',        'Iqbal',     'M', 'asif.iqbal.gulshan@gmail.com',      '+8801711093846', 'House 6, Road 90, Gulshan 2',              'Dhaka',              'Dhaka',        'Dhaka',      'O+',  'Entrepreneur',             4, true,  true,  150,  1,    390, NULL),
('Shirin',      'Sultana',   'F', 'shirin.sultana@gmail.com',          '+8801818725046', 'Palash Bazar',                             'Palash',             'Narsingdi',    'Dhaka',      'B+',  'Teacher',                  4, true,  false, 280,  15,   300, 2),
('Rakibul',     'Hasan',     'M', 'rakibul.hasan.ju@gmail.com',        '+8801521874309', 'Al Beruni Hall, Jahangirnagar University', 'Savar',              'Dhaka',        'Dhaka',      'A+',  'Student',                  4, true,  false, 20,   0,    140, NULL),
('Sanjida',     'Haque',     'F', 'sanjida.haque@gmail.com',           '+8801760912834', '27 Rankin Street, Wari',                   'Dhaka',              'Dhaka',        'Dhaka',      'B+',  'Software Engineer',        4, true,  false, 88,   1,    240, NULL),
-- ===== Chattogram division =====
('Saiful',      'Alam',      'M', 'saiful.alam.ctg@gmail.com',         '+8801815243697', 'House 41, Road 2, Khulshi',                'Chattogram',         'Chattogram',   'Chattogram', 'B+',  'Shipping Executive',       4, true,  false, 132,  2,    440, NULL),
('Tasnim',      'Chowdhury', 'F', 'tasnim.chowdhury@gmail.com',        '+8801716458932', 'Road 4, Agrabad C/A',                      'Chattogram',         'Chattogram',   'Chattogram', 'O+',  'Doctor',                   4, true,  true,  210,  5,    350, NULL),
('Jahidul',     'Islam',     'M', 'jahidul.islam.ctg@gmail.com',       '+8801633478215', 'H Block, Halishahar',                      'Chattogram',         'Chattogram',   'Chattogram', 'O+',  'Garments Supervisor',      4, true,  false, 85,   3,    310, NULL),
('Nadia',       'Sultana',   'F', 'nadia.sultana.cox@gmail.com',       '+8801873256140', 'Kolatoli Road',                            'Cox''s Bazar Sadar', 'Cox''s Bazar', 'Chattogram', 'A+',  'Hotel Manager',            4, true,  false, NULL, 11,   160, NULL),
('Monirul',     'Islam',     'M', 'monirul.islam.cumilla@gmail.com',   '+8801914583720', 'Kandirpar',                                'Comilla Sadar',      'Comilla',      'Chattogram', 'B+',  'Bank Officer',             4, true,  false, 240,  8,    395, NULL),
('Shahana',     'Parvin',    'F', 'shahana.parvin@gmail.com',          '+8801745129836', 'Trunk Road',                               'Feni Sadar',         'Feni',         'Chattogram', 'AB+', 'Nurse',                    4, true,  false, 110,  2,    270, 1),
('Ariful',      'Haque',     'M', 'ariful.haque.noakhali@gmail.com',   '+8801827364519', 'Maijdee Court',                            'Noakhali Sadar',     'Noakhali',     'Chattogram', 'O-',  'Civil Engineer',           4, true,  false, 175,  6,    330, NULL),
('Rehana',      'Begum',     'F', 'rehana.begum.bb@gmail.com',         '+8801558423796', 'Court Road',                               'Brahmanbaria Sadar', 'Brahmanbaria', 'Chattogram', 'B+',  'Teacher',                  4, true,  true,  NULL, 25,   200, NULL),
('Fahim',       'Uddin',     'M', 'fahim.uddin.cuet@gmail.com',        '+8801684217359', 'CUET Campus, Pahartali',                   'Raozan',             'Chattogram',   'Chattogram', 'A+',  'Student',                  4, true,  false, 55,   1,    230, NULL),
('Mithila',     'Das',       'F', 'mithila.das@gmail.com',             '+8801932145786', 'Patharghata',                              'Chattogram',         'Chattogram',   'Chattogram', 'O+',  'Pharmacist',               4, true,  false, 290,  9,    380, NULL),
('Kawsar',      'Ahmed',     'M', 'kawsar.ahmed.chandpur@gmail.com',   '+8801719836542', 'Kalibari',                                 'Chandpur Sadar',     'Chandpur',     'Chattogram', 'B-',  'Businessman',              4, true,  false, 360,  14,   450, NULL),
-- ===== Rajshahi division =====
('Habibur',     'Rahman',    'M', 'habibur.rahman.rj@gmail.com',       '+8801711564823', 'Shaheb Bazar, Boalia',                     'Rajshahi',           'Rajshahi',     'Rajshahi',   'B+',  'University Lecturer',      4, true,  false, 145,  3,    400, NULL),
('Sharmin',     'Akter',     'F', 'sharmin.akter.ru@gmail.com',        '+8801778341265', 'Binodpur, Motihar',                        'Rajshahi',           'Rajshahi',     'Rajshahi',   'O+',  'Student',                  4, true,  false, 70,   1,    190, NULL),
('Mostafizur',  'Rahman',    'M', 'mostafiz.bogura@gmail.com',         '+8801923654178', 'Satmatha',                                 'Bogra Sadar',        'Bogra',        'Rajshahi',   'A+',  'Businessman',              4, true,  false, 220,  10,   340, NULL),
('Lipi',        'Khatun',    'F', 'lipi.khatun@gmail.com',             '+8801540296385', 'Radhanagar',                               'Pabna Sadar',        'Pabna',        'Rajshahi',   'O+',  'Nurse',                    4, true,  true,  NULL, 18,   150, NULL),
('Abdullah Al', 'Mamun',     'M', 'almamun.sirajganj@gmail.com',       '+8801812745390', 'S.S. Road',                                'Sirajganj Sadar',    'Sirajganj',    'Rajshahi',   'AB-', 'Doctor',                   4, true,  false, 190,  5,    360, 2),
('Rubel',       'Hossain',   'M', 'rubel.hossain.natore@gmail.com',    '+8801686153274', 'Kanaikhali',                               'Natore Sadar',       'Natore',       'Rajshahi',   'B+',  'Agricultural Officer',     4, true,  false, 100,  7,    280, NULL),
-- ===== Khulna division =====
('Sohel',       'Rana',      'M', 'sohel.rana.khulna@gmail.com',       '+8801716352948', 'KDA Avenue, Sonadanga',                    'Khulna',             'Khulna',       'Khulna',     'O+',  'Mechanical Engineer',      4, true,  false, 125,  2,    410, NULL),
('Moumita',     'Saha',      'F', 'moumita.saha@gmail.com',            '+8801834567129', 'Ghop Road',                                'Jessore Sadar',      'Jessore',      'Khulna',     'A+',  'Teacher',                  4, true,  false, 250,  8,    330, NULL),
('Ashraful',    'Alam',      'M', 'ashraful.alam.kushtia@gmail.com',   '+8801957283614', 'NS Road',                                  'Kushtia Sadar',      'Kushtia',      'Khulna',     'B+',  'Journalist',               4, true,  false, 80,   4,    250, NULL),
('Rokeya',      'Khanam',    'F', 'rokeya.khanam@gmail.com',           '+8801559374812', 'Port Area',                                'Mongla',             'Bagerhat',     'Khulna',     'O+',  'Homemaker',                4, false, true,  NULL, NULL, 45,   NULL),
('Biplob Kumar','Ghosh',     'M', 'biplob.ghosh@gmail.com',            '+8801723458196', 'Shyamnagar Bazar',                         'Shyamnagar',         'Satkhira',     'Khulna',     'AB+', 'Businessman',              4, true,  false, 310,  12,   420, NULL),
-- ===== Sylhet division =====
('Rezaul',      'Karim',     'M', 'rezaul.karim.syl@gmail.com',        '+8801711875432', 'Ambarkhana',                               'Sylhet Sadar',       'Sylhet',       'Sylhet',     'O+',  'Bank Officer',             4, true,  false, 160,  3,    370, NULL),
('Fatema Tuz',  'Zohra',     'F', 'fatema.zohra@gmail.com',            '+8801818963425', 'Block C, Uposhohor',                       'Sylhet Sadar',       'Sylhet',       'Sylhet',     'B+',  'Doctor',                   4, true,  true,  200,  5,    320, NULL),
('Jubayer',     'Ahmed',     'M', 'jubayer.ahmed.sust@gmail.com',      '+8801639427518', 'SUST Campus, Akhalia',                     'Sylhet Sadar',       'Sylhet',       'Sylhet',     'A-',  'Student',                  4, true,  false, 35,   0,    120, NULL),
('Anwar',       'Hossain',   'M', 'anwar.hossain.srimangal@gmail.com', '+8801917245863', 'Station Road',                             'Sreemangal',         'Moulvibazar',  'Sylhet',     'B+',  'Tea Garden Manager',       4, true,  false, 275,  16,   390, NULL),
-- ===== Rangpur division =====
('Mizanur',     'Rahman',    'M', 'mizanur.rahman.rangpur@gmail.com',  '+8801714932657', 'Jahaj Company More',                       'Rangpur Sadar',      'Rangpur',      'Rangpur',    'B+',  'Govt. Officer',            4, true,  false, 140,  4,    360, NULL),
('Shapla',      'Begum',     'F', 'shapla.begum@gmail.com',            '+8801785263419', 'Balubari',                                 'Dinajpur Sadar',     'Dinajpur',     'Rangpur',    'O+',  'Teacher',                  4, true,  false, 320,  20,   300, NULL),
('Nayeem',      'Islam',     'M', 'nayeem.islam.saidpur@gmail.com',    '+8801529384716', 'Cantonment Road',                          'Saidpur',            'Nilphamari',   'Rangpur',    'A+',  'Railway Employee',         4, true,  false, 60,   2,    170, NULL),
-- ===== Barisal division =====
('Kamal',       'Uddin',     'M', 'kamal.uddin.bsl@gmail.com',         '+8801712658349', 'Sadar Road',                               'Barisal Sadar',      'Barisal',      'Barisal',    'O+',  'Businessman',              4, true,  false, 185,  6,    380, NULL),
('Taslima',     'Akter',     'F', 'taslima.akter.bsl@gmail.com',       '+8801846239157', 'Band Road',                                'Barisal Sadar',      'Barisal',      'Barisal',    'B+',  'Nurse',                    4, true,  false, 95,   3,    260, NULL),
('Hasan',       'Mahmud',    'M', 'hasan.mahmud.bhola@gmail.com',      '+8801963857241', 'Sadar Road',                               'Bhola Sadar',        'Bhola',        'Barisal',    'AB+', 'Fisheries Officer',        4, true,  false, NULL, 9,    130, NULL),
('Sharif',      'Hossain',   'M', 'sharif.hossain.ptk@gmail.com',      '+8801671538294', 'Kalapara Bazar',                           'Kalapara',           'Patuakhali',   'Barisal',    'O+',  'Teacher',                  4, true,  true,  230,  11,   340, NULL),
-- ===== Mymensingh division =====
('Rashed',      'Khan',      'M', 'rashed.khan.mym@gmail.com',         '+8801715623498', 'Ganginarpar',                              'Mymensingh Sadar',   'Mymensingh',   'Mymensingh', 'B+',  'Doctor',                   4, true,  false, 115,  2,    400, NULL),
('Mousumi',     'Akter',     'F', 'mousumi.akter.bau@gmail.com',       '+8801827594163', 'BAU Campus',                               'Mymensingh Sadar',   'Mymensingh',   'Mymensingh', 'A+',  'Student',                  4, true,  false, 50,   1,    210, NULL),
('Rafsan',      'Jani',      'M', 'rafsan.jani@gmail.com',             '+8801542768391', 'Station Road',                             'Jamalpur Sadar',     'Jamalpur',     'Mymensingh', 'O+',  'Bank Officer',             4, true,  false, 265,  13,   350, NULL)
) AS v(fn, ln, g, email, phone, addr, city, district, division, bg, prof,
       role_id, verified, hide, donated, login, created, created_by);

INSERT INTO public.users (
    first_name, last_name, gender, email, password, phone, address,
    city, district, division, country, blood_group, date_of_birth, profession, role_id,
    is_available, is_verified, hide_personal_info, is_password_temporary,
    donated_at, last_login, created_at, updated_at, created_by, verification_code
)
SELECT
    v.fn, v.ln, v.g, v.email,
    '$2a$10$JN6X.THRe8nrKzpWeY4Xrupd3Y4tKvVEHrmF8IsjyZxgBn0pU9Ake',
    v.phone, v.addr, v.city, v.district, v.division, 'Bangladesh',
    v.bg,
    -- deterministic age: students 19–23, everyone else 24–49
    (current_date - make_interval(
        years => CASE WHEN v.prof = 'Student' THEN 19 + abs(hashtext(v.email)) % 5
                      ELSE 24 + abs(hashtext(v.email)) % 26 END,
        days  => abs(hashtext(v.fn || v.ln)) % 365))::date,
    v.prof, v.role_id,
    true, v.verified, v.hide, false,
    CASE WHEN v.donated IS NULL THEN NULL
         ELSE date_trunc('day', localtimestamp) - make_interval(days => v.donated) + interval '10 hours 30 minutes' END,
    CASE WHEN v.login IS NULL THEN NULL
         ELSE localtimestamp - make_interval(days => v.login, hours => length(v.email) % 9) END,
    localtimestamp - make_interval(days => v.created, hours => length(v.fn) % 12),
    localtimestamp - make_interval(days => LEAST(v.created, COALESCE(v.login, v.created))),
    v.created_by,
    md5(random()::text || v.email)
FROM seed_users v
ORDER BY v.created DESC
ON CONFLICT (email) DO NOTHING;


-- =====================================================================
-- 5. USER LOCATIONS (GeoNames ids of each division's main city)
-- =====================================================================
INSERT INTO public.user_locations (country, country_geo_id, city, city_geo_id, is_trusted, user_id, trust_code, created_at)
SELECT 'Bangladesh', 1210997, g.city, g.geo_id, true, u.id, md5(random()::text || u.id), u.created_at
FROM seed_users s
JOIN public.users u ON u.email = s.email
JOIN (VALUES
    ('Dhaka',      'Dhaka',      1185241),
    ('Chattogram', 'Chittagong', 1205733),
    ('Khulna',     'Khulna',     1336135),
    ('Rajshahi',   'Rajshahi',   1185128),
    ('Sylhet',     'Sylhet',     1185099),
    ('Rangpur',    'Rangpur',    1185188),
    ('Barisal',    'Barisal',    1336137),
    ('Mymensingh', 'Mymensingh', 1185166)
) AS g(division, city, geo_id) ON g.division = u.division
WHERE u.last_login IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM public.user_locations ul WHERE ul.user_id = u.id);

-- A few untrusted logins from new places (travel / new device)
INSERT INTO public.user_locations (country, country_geo_id, city, city_geo_id, is_trusted, user_id, trust_code, created_at)
SELECT v.country, v.country_geo, v.city, v.city_geo, false, u.id, md5(random()::text || u.id || v.city),
       localtimestamp - make_interval(days => v.days_ago)
FROM (VALUES
    ('tanvir.ahmed92@gmail.com',       'Bangladesh',           1210997, 'Chittagong', 1205733, 34),
    ('asif.iqbal.gulshan@gmail.com',   'Singapore',            1880251, 'Singapore',  1880252, 62),
    ('saiful.alam.ctg@gmail.com',      'Bangladesh',           1210997, 'Dhaka',      1185241, 18),
    ('rezaul.karim.syl@gmail.com',     'United Arab Emirates', 290557,  'Dubai',      292223,  90),
    ('kamrul.hasan@blood-link.online', 'Bangladesh',           1210997, 'Khulna',     1336135, 7)
) AS v(email, country, country_geo, city, city_geo, days_ago)
JOIN public.users u ON u.email = v.email
WHERE NOT EXISTS (
    SELECT 1 FROM public.user_locations ul
    WHERE ul.user_id = u.id AND ul.city_geo_id = v.city_geo
);


-- =====================================================================
-- 6. SESSIONS: historical and already expired (is_valid = false,
--    expire_at in the past), so none of them can authenticate.
-- =====================================================================
INSERT INTO public.sessions (token, role_id, user_id, loggedin_profile_id, is_valid, last_used, created_at, currency, expire_at, iso_country)
SELECT md5(random()::text || u.id || v.started) || md5(clock_timestamp()::text || u.email),
       u.role_id, u.id, NULL, false,
       localtimestamp - make_interval(days => v.started) + interval '2 hours',
       localtimestamp - make_interval(days => v.started),
       'BDT',
       localtimestamp - make_interval(days => v.started) + interval '1 day',
       'BD'
FROM (VALUES
    ('kamrul.hasan@blood-link.online',   3), ('kamrul.hasan@blood-link.online', 12),
    ('shamima.nasrin@blood-link.online', 4), ('abu.sayeed@blood-link.online',    5),
    ('tanvir.ahmed92@gmail.com',         2), ('tanvir.ahmed92@gmail.com',       21),
    ('mahmud.hasan.du@gmail.com',        1), ('sadia.islam07@gmail.com',         6),
    ('imran.kabir.mpur@gmail.com',       2), ('sabbir.rahman@gmail.com',         3),
    ('saiful.alam.ctg@gmail.com',        3), ('fahim.uddin.cuet@gmail.com',      2),
    ('habibur.rahman.rj@gmail.com',      4), ('sohel.rana.khulna@gmail.com',     3),
    ('jubayer.ahmed.sust@gmail.com',     1), ('rashed.khan.mym@gmail.com',       3)
) AS v(email, started)
JOIN public.users u ON u.email = v.email
WHERE NOT EXISTS (
    SELECT 1 FROM public.sessions s
    WHERE s.user_id = u.id
      AND s.created_at::date = (localtimestamp - make_interval(days => v.started))::date
);


-- =====================================================================
-- 7. DONATION REQUESTS
--    req_offset = required date relative to today (NULL for completed:
--                 then the date is the donor's actual last donation date)
--    lead       = days between the request being created and required
-- =====================================================================
INSERT INTO public.donation_requests (
    requester_id, donor_id, blood_group, urgency, patient_name, reason, units_needed,
    hospital_name, address, district, division, contact_phone, required_date,
    status, is_admin_request, created_at, updated_at, created_by, updated_by
)
SELECT r.id, d.id, v.bg, v.urgency, v.patient, v.reason, v.units,
       v.hospital, v.addr, v.district, v.division, v.phone, x.req_date,
       v.status, v.is_admin, c.created_at,
       CASE WHEN v.status = 'pending' THEN c.created_at
            ELSE LEAST(localtimestamp, c.created_at + interval '5 hours') END,
       r.id,
       CASE WHEN v.status IN ('accepted','rejected') THEN d.id ELSE r.id END
FROM (VALUES
-- ----- completed (donated on the donor's last donation date) -----
('nusrat.jahan@yahoo.com',          'tanvir.ahmed92@gmail.com',        'B+',  'urgent',    'Abdul Mannan',        'Open heart surgery (CABG)',               2, 'National Institute of Cardiovascular Diseases',    'Sher-e-Bangla Nagar',                            'Dhaka',       'Dhaka',      '+8801819456732', NULL, 'completed', false, 3),
('sumaiya.rahman@gmail.com',        'sadia.islam07@gmail.com',         'B+',  'normal',    'Rahima Khatun',       'Caesarean section',                       1, 'Labaid Specialized Hospital',                      'House 1, Road 4, Dhanmondi',                     'Dhaka',       'Dhaka',      '+8801782654310', NULL, 'completed', false, 5),
('kamrul.hasan@blood-link.online',  'rafiqul.islam.savar@gmail.com',   'O+',  'emergency', 'Md. Jalal Uddin',     'Road accident — internal bleeding',       3, 'Enam Medical College Hospital',                    'Thana Road, Savar',                              'Dhaka',       'Dhaka',      '+8801711458236', NULL, 'completed', true,  1),
('asif.iqbal.gulshan@gmail.com',    'imran.kabir.mpur@gmail.com',      'O-',  'emergency', 'Shireen Akhter',      'Postpartum haemorrhage',                  2, 'Evercare Hospital Dhaka',                          'Plot 81, Block E, Bashundhara R/A',              'Dhaka',       'Dhaka',      '+8801711093846', NULL, 'completed', false, 1),
('rumana.afroz@gmail.com',          'nazmul.huda.fp@gmail.com',        'AB+', 'urgent',    'Nurjahan Begum',      'Hip replacement surgery',                 1, 'Faridpur Medical College Hospital',                'Jhiltuli',                                       'Faridpur',    'Dhaka',      '+8801632894175', NULL, 'completed', false, 4),
('mithila.das@gmail.com',           'saiful.alam.ctg@gmail.com',       'B+',  'urgent',    'Subrata Das',         'Gall bladder surgery',                    1, 'Chattogram Medical College Hospital',              'K.B. Fazlul Kader Road, Panchlaish',             'Chattogram',  'Chattogram', '+8801932145786', NULL, 'completed', false, 3),
('shamima.nasrin@blood-link.online','tasnim.chowdhury@gmail.com',      'O+',  'emergency', 'Baby of Farzana',     'Neonatal exchange transfusion',           1, 'Chittagong Maa-O-Shishu Hospital',                 'Agrabad',                                        'Chattogram',  'Chattogram', '+8801819264573', NULL, 'completed', true,  1),
('shahana.parvin@gmail.com',        'monirul.islam.cumilla@gmail.com', 'B+',  'normal',    'Abul Kashem',         'Thalassemia transfusion',                 1, 'Comilla Medical College Hospital',                 'Kuchaitoli',                                     'Comilla',     'Chattogram', '+8801745129836', NULL, 'completed', false, 6),
('sharmin.akter.ru@gmail.com',      'habibur.rahman.rj@gmail.com',     'B+',  'urgent',    'Momena Khatun',       'Uterine surgery',                         2, 'Rajshahi Medical College Hospital',                'Laxmipur',                                       'Rajshahi',    'Rajshahi',   '+8801778341265', NULL, 'completed', false, 2),
('ashraful.alam.kushtia@gmail.com', 'sohel.rana.khulna@gmail.com',     'O+',  'urgent',    'Shahidul Islam',      'Kidney transplant',                       2, 'Khulna Medical College Hospital',                  'Boyra',                                          'Khulna',      'Khulna',     '+8801957283614', NULL, 'completed', false, 4),
('abu.sayeed@blood-link.online',    'rezaul.karim.syl@gmail.com',      'O+',  'emergency', 'Ruksana Begum',       'Severe anaemia during pregnancy',         1, 'Sylhet MAG Osmani Medical College Hospital',       'Kajolshah',                                      'Sylhet',      'Sylhet',     '+8801912384756', NULL, 'completed', true,  1),
('nayeem.islam.saidpur@gmail.com',  'mizanur.rahman.rangpur@gmail.com','B+',  'normal',    'Golam Mostafa',       'Prostate surgery',                        1, 'Rangpur Medical College Hospital',                 'Dhap',                                           'Rangpur',     'Rangpur',    '+8801529384716', NULL, 'completed', false, 5),
('mousumi.akter.bau@gmail.com',     'rashed.khan.mym@gmail.com',       'B+',  'urgent',    'Hosneara Begum',      'Blood cancer — chemotherapy support',     2, 'Mymensingh Medical College Hospital',              'Charpara',                                       'Mymensingh',  'Mymensingh', '+8801827594163', NULL, 'completed', false, 3),
-- ----- accepted (donor confirmed, date coming up) -----
('sabbir.rahman@gmail.com',         'rumana.afroz@gmail.com',          'A+',  'urgent',    'Shamsun Nahar',       'Knee replacement surgery',                1, 'Square Hospital',                                  '18/F Bir Uttam Qazi Nuruzzaman Sarak, Panthapath','Dhaka',      'Dhaka',      '+8801974512368', 3,    'accepted',  false, 4),
('sanjida.haque@gmail.com',         'shirin.sultana@gmail.com',        'B+',  'normal',    'Anwara Begum',        'Hysterectomy',                            1, 'Holy Family Red Crescent Medical College Hospital','1 Eskaton Garden Road',                          'Dhaka',       'Dhaka',      '+8801760912834', 5,    'accepted',  false, 7),
('jahidul.islam.ctg@gmail.com',     'mithila.das@gmail.com',           'O+',  'urgent',    'Nurul Absar',         'Appendectomy',                            1, 'Park View Hospital',                               'Panchlaish',                                     'Chattogram',  'Chattogram', '+8801633478215', 2,    'accepted',  false, 3),
('lipi.khatun@gmail.com',           'mostafiz.bogura@gmail.com',       'A+',  'normal',    'Rahela Khatun',       'Caesarean section',                       1, 'Shaheed Ziaur Rahman Medical College Hospital',    'Silimpur',                                       'Bogra',       'Rajshahi',   '+8801540296385', 4,    'accepted',  false, 6),
('fatema.zohra@gmail.com',          'anwar.hossain.srimangal@gmail.com','B+', 'urgent',    'Joynal Abedin',       'Gastric bleeding',                        2, 'Sylhet MAG Osmani Medical College Hospital',       'Kajolshah',                                      'Sylhet',      'Sylhet',     '+8801818963425', 1,    'accepted',  false, 2),
('taslima.akter.bsl@gmail.com',     'kamal.uddin.bsl@gmail.com',       'O+',  'normal',    'Monowara Begum',      'Thyroid surgery',                         1, 'Sher-e-Bangla Medical College Hospital',           'Band Road',                                      'Barisal',     'Barisal',    '+8801846239157', 6,    'accepted',  false, 8),
-- ----- rejected (donor had donated too recently) -----
('tanvir.ahmed92@gmail.com',        'mahmud.hasan.du@gmail.com',       'A+',  'urgent',    'Kulsum Begum',        'Dengue — platelet transfusion',           1, 'Dhaka Medical College Hospital',                   'Bakshibazar',                                    'Dhaka',       'Dhaka',      '+8801712345981', -6,   'rejected',  false, 2),
('rezaul.karim.syl@gmail.com',      'jubayer.ahmed.sust@gmail.com',    'A-',  'urgent',    'Habiba Sultana',      'Caesarean section',                       1, 'Sylhet MAG Osmani Medical College Hospital',       'Kajolshah',                                      'Sylhet',      'Sylhet',     '+8801711875432', -10,  'rejected',  false, 3),
('ayesha.siddika@gmail.com',        'mehedi.hasan.bhairab@gmail.com',  'B+',  'normal',    'Fazlul Haque',        'Hernia surgery',                          1, 'Shahid Syed Nazrul Islam Medical College Hospital','Kishoreganj Sadar',                              'Kishoreganj', 'Dhaka',      '+8801923478156', -15,  'rejected',  false, 4),
-- ----- pending, sent to a specific donor -----
('arif.hossain.bd@outlook.com',     'farhana.akter88@gmail.com',       'AB+', 'urgent',    'Rowshan Ara',         'Ovarian cyst surgery',                    1, 'Shaheed Tajuddin Ahmad Medical College Hospital',  'Gazipur Sadar',                                  'Gazipur',     'Dhaka',      '+8801556783412', 2,    'pending',   false, 3),
('ariful.haque.noakhali@gmail.com', 'kawsar.ahmed.chandpur@gmail.com', 'B-',  'urgent',    'Delwar Hossain',      'Bypass surgery',                          2, 'Chattogram Medical College Hospital',              'K.B. Fazlul Kader Road, Panchlaish',             'Chattogram',  'Chattogram', '+8801827364519', 4,    'pending',   false, 5),
('moumita.saha@gmail.com',          'biplob.ghosh@gmail.com',          'AB+', 'normal',    'Gopal Chandra Saha',  'Kidney dialysis',                         1, 'Khulna Medical College Hospital',                  'Boyra',                                          'Khulna',      'Khulna',     '+8801834567129', 6,    'pending',   false, 8),
('mizanur.rahman.rangpur@gmail.com','shapla.begum@gmail.com',          'O+',  'normal',    'Sufia Khatun',        'Fibroid removal surgery',                 1, 'Rangpur Medical College Hospital',                 'Dhap',                                           'Rangpur',     'Rangpur',    '+8801714932657', 5,    'pending',   false, 6),
('rashed.khan.mym@gmail.com',       'rafsan.jani@gmail.com',           'O+',  'urgent',    'Azizul Haque',        'Road accident — leg fracture',            2, 'Mymensingh Medical College Hospital',              'Charpara',                                       'Mymensingh',  'Mymensingh', '+8801715623498', 1,    'pending',   false, 2),
-- ----- pending, open to all donors -----
('admin@blood-link.online',         NULL,                              'O-',  'emergency', 'Shahnaz Parvin',      'Placenta previa — emergency C-section',   3, 'Dhaka Medical College Hospital',                   'Bakshibazar',                                    'Dhaka',       'Dhaka',      '+8801711458236', 0,    'pending',   true,  0),
('kamrul.hasan@blood-link.online',  NULL,                              'AB-', 'emergency', 'Md. Shafiqul Islam',  'Liver surgery',                           2, 'Rajshahi Medical College Hospital',                'Laxmipur',                                       'Rajshahi',    'Rajshahi',   '+8801711458236', 1,    'pending',   true,  1),
('shamima.nasrin@blood-link.online',NULL,                              'A+',  'urgent',    'Tahmid Hasan (8 yrs)','Thalassemia — monthly transfusion',       1, 'Chattogram Medical College Hospital',              'K.B. Fazlul Kader Road, Panchlaish',             'Chattogram',  'Chattogram', '+8801819264573', 3,    'pending',   true,  4),
('sadia.islam07@gmail.com',         NULL,                              'B-',  'urgent',    'Rabeya Akter',        'Caesarean section',                       1, 'Ibn Sina Hospital',                                'House 48, Road 9/A, Dhanmondi',                  'Dhaka',       'Dhaka',      '+8801534872910', 2,    'pending',   false, 3),
('mahmud.hasan.du@gmail.com',       NULL,                              'O+',  'urgent',    'Nafisa Tabassum',     'Dengue — platelet transfusion',           1, 'Mugda Medical College Hospital',                   'Mugda',                                          'Dhaka',       'Dhaka',      '+8801674239815', 1,    'pending',   false, 1),
('abu.sayeed@blood-link.online',    NULL,                              'A-',  'normal',    'Kohinoor Begum',      'Hip fracture surgery',                    1, 'North East Medical College Hospital',              'South Surma',                                    'Sylhet',      'Sylhet',     '+8801912384756', 7,    'pending',   true,  9),
-- ===== TEST SCENARIOS ==============================================
-- Requests sent TO the test account donor@blood-link.online, one per status
('nusrat.jahan@yahoo.com',          'donor@blood-link.online',         'B+',  'emergency', 'Delwara Begum',       'Upper GI bleeding',                       2, 'Dhaka Medical College Hospital',                   'Bakshibazar',                                    'Dhaka',       'Dhaka',      '+8801819456732', 1,    'pending',   false, 1),
('rumana.afroz@gmail.com',          'donor@blood-link.online',         'B+',  'normal',    'Shahida Parvin',      'Hysterectomy',                            1, 'Labaid Specialized Hospital',                      'House 1, Road 4, Dhanmondi',                     'Dhaka',       'Dhaka',      '+8801632894175', 6,    'pending',   false, 2),
('sabbir.rahman@gmail.com',         'donor@blood-link.online',         'B+',  'urgent',    'Moniruzzaman Khan',   'Heart valve replacement',                 2, 'Ibrahim Cardiac Hospital & Research Institute',    '122 Kazi Nazrul Islam Avenue, Shahbag',          'Dhaka',       'Dhaka',      '+8801974512368', 3,    'accepted',  false, 4),
('asif.iqbal.gulshan@gmail.com',    'donor@blood-link.online',         'B+',  'normal',    'Ferdousi Begum',      'Knee arthroscopy',                        1, 'Square Hospital',                                  '18/F Bir Uttam Qazi Nuruzzaman Sarak, Panthapath','Dhaka',      'Dhaka',      '+8801711093846', -4,   'rejected',  false, 3),
('kamrul.hasan@blood-link.online',  'donor@blood-link.online',         'B+',  'emergency', 'Abdul Halim',         'Road accident — head injury',             1, 'Dhaka Medical College Hospital',                   'Bakshibazar',                                    'Dhaka',       'Dhaka',      '+8801711458236', NULL, 'completed', true,  1),
-- Requests made BY the test account, one per status
('donor@blood-link.online',         NULL,                              'B+',  'urgent',    'Rokshana Akter',      'Caesarean section',                       1, 'Ibn Sina Hospital',                                'House 48, Road 9/A, Dhanmondi',                  'Dhaka',       'Dhaka',      '+8801700112233', 2,    'pending',   false, 1),
('donor@blood-link.online',         'imran.kabir.mpur@gmail.com',      'O-',  'urgent',    'Hafiz Uddin',         'Bypass surgery',                          2, 'National Institute of Cardiovascular Diseases',    'Sher-e-Bangla Nagar',                            'Dhaka',       'Dhaka',      '+8801700112233', 5,    'pending',   false, 2),
('donor@blood-link.online',         'sabbir.rahman@gmail.com',         'B+',  'normal',    'Jahanara Begum',      'Kidney dialysis',                         1, 'Kidney Foundation Hospital',                       'Plot 6, Road 3, Section 2, Mirpur',              'Dhaka',       'Dhaka',      '+8801700112233', 2,    'accepted',  false, 3),
('donor@blood-link.online',         'mehedi.hasan.bhairab@gmail.com',  'B+',  'normal',    'Motaleb Mia',         'Hernia surgery',                          1, 'Dhaka Medical College Hospital',                   'Bakshibazar',                                    'Dhaka',       'Dhaka',      '+8801700112233', -2,   'rejected',  false, 3),
('donor@blood-link.online',         'asif.iqbal.gulshan@gmail.com',    'O+',  'urgent',    'Selina Akhter',       'Gall bladder surgery',                    1, 'Popular Medical College Hospital',                 'House 25, Road 2, Dhanmondi',                    'Dhaka',       'Dhaka',      '+8801700112233', NULL, 'completed', false, 2),
-- Edge cases
('tahmina.akhter@yahoo.com',        NULL,                              'O+',  'normal',    'Nurul Islam',         'Severe anaemia',                          1, 'Munshiganj 250 Bed General Hospital',              'Munshiganj Sadar',                               'Munshiganj',  'Dhaka',      '+8801552348761', -3,   'pending',   false, 2),  -- overdue but still pending
('shakil.ahmed.ng@gmail.com',       NULL,                              'O+',  'urgent',    'Kulsum Bibi',         'Caesarean section',                       1, 'Narayanganj 300 Bed Hospital',                     'Khanpur',                                        'Narayanganj', 'Dhaka',      '+8801816347290', 3,    'pending',   false, 1),  -- soft-deleted below
('admin@blood-link.online',         NULL,                              'AB+', 'emergency', 'Md. Abdus Salam',     'Liver transplant',                        6, 'Evercare Hospital Dhaka',                          'Plot 81, Block E, Bashundhara R/A',              'Dhaka',       'Dhaka',      '+8801711458236', 2,    'pending',   true,  1),  -- large unit count
('superadmin@blood-link.online',    NULL,                              'A-',  'emergency', 'Rasel Mia',           'Severe burn injury (40%)',                4, 'National Institute of Burn and Plastic Surgery',   'Bakshibazar',                                    'Dhaka',       'Dhaka',      '+8801711458236', 0,    'pending',   true,  0),  -- needed today
('jannatul.ferdous99@gmail.com',    NULL,                              'A+',  'normal',    'Afroza Begum',        'Caesarean section',                       1, 'Holy Family Red Crescent Medical College Hospital','1 Eskaton Garden Road',                          'Dhaka',       'Dhaka',      '+8801687453201', 4,    'pending',   false, 1),  -- unverified requester
('nazmul.huda.fp@gmail.com',        NULL,                              'AB+', 'normal',    'Hamida Begum',        'Scheduled hip replacement',               2, 'Faridpur Medical College Hospital',                'Jhiltuli',                                       'Faridpur',    'Dhaka',      '+8801716783429', 30,   'pending',   false, 3),  -- a month ahead
('rakibul.hasan.ju@gmail.com',      'nusrat.jahan@yahoo.com',          'O+',  'urgent',    'Monjur Alam',         'Dengue shock syndrome',                   1, 'Enam Medical College Hospital',                    'Thana Road, Savar',                              'Dhaka',       'Dhaka',      '+8801521874309', 1,    'pending',   false, 1)   -- donor hides personal info
) AS v(requester, donor, bg, urgency, patient, reason, units, hospital, addr,
       district, division, phone, req_offset, status, is_admin, lead)
JOIN public.users r ON r.email = v.requester
LEFT JOIN public.users d ON d.email = v.donor
CROSS JOIN LATERAL (
    SELECT COALESCE(current_date + v.req_offset, d.donated_at::date) AS req_date
) x
CROSS JOIN LATERAL (
    SELECT LEAST(localtimestamp - interval '20 minutes',
                 x.req_date - make_interval(days => v.lead)
                 + interval '8 hours' + make_interval(mins => abs(hashtext(v.patient)) % 600)
           )::timestamp AS created_at
) c
WHERE x.req_date IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM public.donation_requests dr
    WHERE dr.requester_id = r.id AND dr.patient_name = v.patient
)
ORDER BY c.created_at;

-- Edge case: a request the requester cancelled (soft-deleted, must not be listed)
UPDATE public.donation_requests dr
SET is_deleted = true, updated_at = dr.created_at + interval '3 hours'
FROM public.users u
WHERE u.id = dr.requester_id
  AND u.email = 'shakil.ahmed.ng@gmail.com'
  AND dr.patient_name = 'Kulsum Bibi'
  AND dr.is_deleted = false;


-- =====================================================================
-- 8. DONATION POSTS
-- =====================================================================
INSERT INTO public.donation_posts (
    author_id, blood_group, title, description, hospital_name, address,
    district, division, units_needed, contact_phone, required_date, status,
    created_at, updated_at
)
SELECT u.id, v.bg, v.title, v.descr, v.hospital, v.addr, v.district, v.division,
       v.units, v.phone, current_date + v.req_offset, v.status,
       localtimestamp - make_interval(days => v.created_ago, mins => abs(hashtext(v.title)) % 900),
       CASE WHEN v.status = 'open'
            THEN localtimestamp - make_interval(days => v.created_ago, mins => abs(hashtext(v.title)) % 900)
            ELSE (current_date + v.req_offset) + interval '18 hours' END
FROM (VALUES
-- ----- open -----
('jahidul.islam.ctg@gmail.com',   'B+',  'B+ রক্তের জরুরি প্রয়োজন',
 'আমার মায়ের অপারেশনের জন্য ২ ব্যাগ B+ রক্তের প্রয়োজন। দয়া করে সাহায্য করুন।',
 'Chattogram Medical College Hospital', 'K.B. Fazlul Kader Road, Panchlaish', 'Chattogram', 'Chattogram', 2, '+8801633478215', 2, 'open', 1),
('nusrat.jahan@yahoo.com',        'O-',  'Urgent: O- blood needed for a newborn',
 'A 3-day-old baby in the NICU needs O negative blood for an exchange transfusion. The donor must be healthy and must not have donated in the last 4 months. Please call directly.',
 'Dhaka Shishu Hospital', 'Sher-e-Bangla Nagar', 'Dhaka', 'Dhaka', 1, '+8801819456732', 1, 'open', 0),
('sabbir.rahman@gmail.com',       'A+',  'Need 2 bags A+ for heart surgery',
 'My uncle is scheduled for bypass surgery on Thursday at NICVD. We need 2 bags of A+ blood. Transport support will be provided for donors.',
 'National Institute of Cardiovascular Diseases', 'Sher-e-Bangla Nagar', 'Dhaka', 'Dhaka', 2, '+8801974512368', 4, 'open', 2),
('mithila.das@gmail.com',         'AB+', 'AB+ needed for dialysis patient',
 'My father is on regular dialysis and needs 1 bag of AB+ blood this week. Any healthy donor from Chattogram city, please contact me.',
 'Imperial Hospital', 'Zakir Hossain Road, Pahartali', 'Chattogram', 'Chattogram', 1, '+8801932145786', 5, 'open', 1),
('habibur.rahman.rj@gmail.com',   'O+',  'ডেঙ্গু রোগীর জন্য প্লাটিলেট প্রয়োজন',
 'রাজশাহী মেডিকেল কলেজ হাসপাতালে ভর্তি একজন ডেঙ্গু রোগীর জন্য জরুরি ভিত্তিতে O+ প্লাটিলেট প্রয়োজন। যোগাযোগ করুন।',
 'Rajshahi Medical College Hospital', 'Laxmipur', 'Rajshahi', 'Rajshahi', 1, '+8801711564823', 1, 'open', 0),
('sohel.rana.khulna@gmail.com',   'B-',  'Rare B- blood needed for C-section',
 'My sister-in-law needs B negative blood for a scheduled caesarean on Saturday. B- is rare — please share this post.',
 'Khulna Medical College Hospital', 'Boyra', 'Khulna', 'Khulna', 1, '+8801716352948', 3, 'open', 1),
('rezaul.karim.syl@gmail.com',    'A+',  'A+ blood for accident victim',
 'A young man injured in a motorbike accident on the Sylhet–Sunamganj highway urgently needs 2 bags of A+ blood.',
 'Sylhet MAG Osmani Medical College Hospital', 'Kajolshah', 'Sylhet', 'Sylhet', 2, '+8801711875432', 0, 'open', 0),
('kamal.uddin.bsl@gmail.com',     'O+',  'থ্যালাসেমিয়া শিশুর জন্য রক্ত প্রয়োজন',
 'আমার ৭ বছরের ভাতিজা থ্যালাসেমিয়া রোগে আক্রান্ত। প্রতি মাসে ১ ব্যাগ O+ রক্ত লাগে। এই মাসের জন্য একজন ডোনার প্রয়োজন।',
 'Sher-e-Bangla Medical College Hospital', 'Band Road', 'Barisal', 'Barisal', 1, '+8801712658349', 6, 'open', 2),
('mousumi.akter.bau@gmail.com',   'B+',  'B+ needed for cancer patient',
 'A chemotherapy patient at MMCH needs B+ blood. Students of BAU and anyone nearby, please come forward.',
 'Mymensingh Medical College Hospital', 'Charpara', 'Mymensingh', 'Mymensingh', 1, '+8801827594163', 2, 'open', 1),
('arif.hossain.bd@outlook.com',   'AB-', 'Looking for an AB- donor in Gazipur',
 'AB negative blood is needed for a liver surgery patient. Any donor from Gazipur or Dhaka, please contact me. Transport and refreshments will be arranged.',
 'Shaheed Tajuddin Ahmad Medical College Hospital', 'Gazipur Sadar', 'Gazipur', 'Dhaka', 1, '+8801556783412', 3, 'open', 1),
-- ----- fulfilled -----
('sadia.islam07@gmail.com',       'O+',  'O+ needed at Square Hospital',
 'We needed 2 bags of O+ for my grandmother''s surgery. Update: we found donors — thank you, everyone!',
 'Square Hospital', '18/F Bir Uttam Qazi Nuruzzaman Sarak, Panthapath', 'Dhaka', 'Dhaka', 2, '+8801534872910', -12, 'fulfilled', 15),
('saiful.alam.ctg@gmail.com',     'A+',  'A+ blood for a delivery patient',
 'A+ blood was needed for my wife''s delivery at Maa-O-Shishu Hospital. Alhamdulillah, a donor from this site helped us.',
 'Chittagong Maa-O-Shishu Hospital', 'Agrabad', 'Chattogram', 'Chattogram', 1, '+8801815243697', -25, 'fulfilled', 27),
('rubel.hossain.natore@gmail.com','B+',  'B+ রক্ত প্রয়োজন - নাটোর',
 'নাটোর সদর হাসপাতালে একজন রোগীর জন্য ১ ব্যাগ B+ রক্ত প্রয়োজন ছিল। ডোনার পাওয়া গেছে, সবাইকে ধন্যবাদ।',
 'Natore Sadar Hospital', 'Natore Sadar', 'Natore', 'Rajshahi', 1, '+8801686153274', -40, 'fulfilled', 42),
('fatema.zohra@gmail.com',        'O+',  'O+ for gastric ulcer bleeding',
 'A patient admitted with GI bleeding needed 3 bags of O+. Everything is arranged now — thanks to the donors.',
 'North East Medical College Hospital', 'South Surma', 'Sylhet', 'Sylhet', 3, '+8801818963425', -60, 'fulfilled', 61),
('tanvir.ahmed92@gmail.com',      'B+',  'B+ needed for my colleague''s father',
 'Need 1 bag of B+ at Ibn Sina Hospital, Dhanmondi, for a surgery tomorrow morning.',
 'Ibn Sina Hospital', 'House 48, Road 9/A, Dhanmondi', 'Dhaka', 'Dhaka', 1, '+8801712345981', -95, 'fulfilled', 96),
('moumita.saha@gmail.com',        'A+',  'A+ for a thalassemia patient in Jessore',
 'Monthly transfusion for a thalassemia patient. Donor found, thank you.',
 'Jessore 250 Bed General Hospital', 'Ghop Road', 'Jessore', 'Khulna', 1, '+8801834567129', -33, 'fulfilled', 35),
-- ----- closed -----
('nadia.sultana.cox@gmail.com',   'O+',  'O+ needed in Cox''s Bazar',
 'Blood was needed for a tourist injured at Kolatoli beach. Closing this post — the patient was transferred to Chattogram.',
 'Cox''s Bazar District Sadar Hospital', 'Hospital Road', 'Cox''s Bazar', 'Chattogram', 2, '+8801873256140', -20, 'closed', 21),
('shakil.ahmed.ng@gmail.com',     'AB+', 'AB+ for surgery (postponed)',
 'The doctor has postponed the surgery, so this request is closed for now.',
 'Narayanganj 300 Bed Hospital', 'Khanpur', 'Narayanganj', 'Dhaka', 1, '+8801816347290', -8, 'closed', 10)
) AS v(author, bg, title, descr, hospital, addr, district, division, units, phone, req_offset, status, created_ago)
JOIN public.users u ON u.email = v.author
WHERE NOT EXISTS (
    SELECT 1 FROM public.donation_posts p WHERE p.author_id = u.id AND p.title = v.title
)
ORDER BY v.created_ago DESC;


-- =====================================================================
-- 9. POST INTERESTS (compatible, nearby donors)
-- =====================================================================
INSERT INTO public.donation_post_interests (post_id, user_id, created_at)
SELECT p.id, u.id,
       LEAST(localtimestamp, p.created_at + make_interval(mins => 15 + abs(hashtext(v.email || v.title)) % 900))
FROM (VALUES
    ('B+ রক্তের জরুরি প্রয়োজন',                      'saiful.alam.ctg@gmail.com'),
    ('B+ রক্তের জরুরি প্রয়োজন',                      'monirul.islam.cumilla@gmail.com'),
    ('B+ রক্তের জরুরি প্রয়োজন',                      'rehana.begum.bb@gmail.com'),
    ('Urgent: O- blood needed for a newborn',         'imran.kabir.mpur@gmail.com'),
    ('Urgent: O- blood needed for a newborn',         'ariful.haque.noakhali@gmail.com'),
    ('Need 2 bags A+ for heart surgery',              'rumana.afroz@gmail.com'),
    ('Need 2 bags A+ for heart surgery',              'rakibul.hasan.ju@gmail.com'),
    ('Need 2 bags A+ for heart surgery',              'jannatul.ferdous99@gmail.com'),
    ('AB+ needed for dialysis patient',               'shahana.parvin@gmail.com'),
    ('ডেঙ্গু রোগীর জন্য প্লাটিলেট প্রয়োজন',            'sharmin.akter.ru@gmail.com'),
    ('ডেঙ্গু রোগীর জন্য প্লাটিলেট প্রয়োজন',            'lipi.khatun@gmail.com'),
    ('Rare B- blood needed for C-section',            'kawsar.ahmed.chandpur@gmail.com'),
    ('A+ blood for accident victim',                  'abu.sayeed@blood-link.online'),
    ('থ্যালাসেমিয়া শিশুর জন্য রক্ত প্রয়োজন',           'sharif.hossain.ptk@gmail.com'),
    ('B+ needed for cancer patient',                  'rashed.khan.mym@gmail.com'),
    ('Looking for an AB- donor in Gazipur',           'almamun.sirajganj@gmail.com'),
    ('O+ needed at Square Hospital',                  'rafiqul.islam.savar@gmail.com'),
    ('O+ needed at Square Hospital',                  'shakil.ahmed.ng@gmail.com'),
    ('O+ needed at Square Hospital',                  'asif.iqbal.gulshan@gmail.com'),
    ('A+ blood for a delivery patient',               'fahim.uddin.cuet@gmail.com'),
    ('A+ blood for a delivery patient',               'nadia.sultana.cox@gmail.com'),
    ('B+ রক্ত প্রয়োজন - নাটোর',                       'habibur.rahman.rj@gmail.com'),
    ('O+ for gastric ulcer bleeding',                 'rezaul.karim.syl@gmail.com'),
    ('B+ needed for my colleague''s father',          'sabbir.rahman@gmail.com'),
    ('B+ needed for my colleague''s father',          'sanjida.haque@gmail.com'),
    ('B+ needed for my colleague''s father',          'sadia.islam07@gmail.com'),
    ('A+ for a thalassemia patient in Jessore',       'mostafiz.bogura@gmail.com'),
    ('O+ needed in Cox''s Bazar',                     'jahidul.islam.ctg@gmail.com'),
    ('O+ needed in Cox''s Bazar',                     'tasnim.chowdhury@gmail.com'),
    ('AB+ for surgery (postponed)',                   'nazmul.huda.fp@gmail.com')
) AS v(title, email)
JOIN public.donation_posts p ON p.title = v.title
JOIN public.users u ON u.email = v.email
ON CONFLICT (post_id, user_id) DO NOTHING;


-- =====================================================================
-- 10. DONATION HISTORY
--     a) every completed request
--     b) each seeded donor's last donation (users.donated_at) not already
--        covered by (a)
--     c) one earlier donation for about two thirds of donors
--     Each donor's newest history row equals users.donated_at.
-- =====================================================================

-- a) completed requests
INSERT INTO public.donation_history (donor_id, recipient_id, recipient_name, request_id, blood_group, units,
                                     hospital_name, district, division, donation_date, created_at)
SELECT dr.donor_id, dr.requester_id, dr.patient_name, dr.id, dr.blood_group, 1,
       dr.hospital_name, dr.district, dr.division, dr.required_date,
       dr.required_date + interval '14 hours'
FROM public.donation_requests dr
JOIN public.users d ON d.id = dr.donor_id
JOIN seed_users s ON s.email = d.email
WHERE dr.status = 'completed'
  AND NOT EXISTS (SELECT 1 FROM public.donation_history h WHERE h.request_id = dr.id)
ORDER BY dr.required_date;

-- b) + c) latest and earlier donations
INSERT INTO public.donation_history (donor_id, recipient_id, recipient_name, blood_group, units,
                                     hospital_name, district, division, donation_date, created_at)
SELECT u.id, NULL, d.recipient, u.blood_group, 1, d.hospital, u.district, u.division,
       d.donation_date, d.donation_date + interval '13 hours'
FROM seed_users s
JOIN public.users u ON u.email = s.email
JOIN (VALUES
    ('Dhaka',      'Dhaka Medical College Hospital'),
    ('Chattogram', 'Chattogram Medical College Hospital'),
    ('Khulna',     'Khulna Medical College Hospital'),
    ('Rajshahi',   'Rajshahi Medical College Hospital'),
    ('Sylhet',     'Sylhet MAG Osmani Medical College Hospital'),
    ('Rangpur',    'Rangpur Medical College Hospital'),
    ('Barisal',    'Sher-e-Bangla Medical College Hospital'),
    ('Mymensingh', 'Mymensingh Medical College Hospital')
) AS h(division, hospital) ON h.division = u.division
CROSS JOIN LATERAL (
    -- b) latest donation, to a patient
    SELECT u.donated_at::date AS donation_date,
           h.hospital,
           (ARRAY['Rokeya Begum','Abdul Karim','Shefali Rani Das','Md. Harun-or-Rashid',
                  'Nasima Akter','Jamal Hossain','Parul Begum','Abdur Rob',
                  'Khadija Khatun','Mofizur Rahman','Anjuman Ara','Sirajul Islam'])
               [1 + abs(hashtext(u.email || 'r')) % 12] AS recipient
    UNION ALL
    -- c) earlier voluntary donation (4–7 months before the latest one)
    SELECT (u.donated_at::date - (120 + abs(hashtext(u.email)) % 90)),
           CASE WHEN u.division = 'Dhaka' THEN 'Bangladesh Red Crescent Blood Centre, Mohammadpur'
                ELSE h.hospital END,
           'Voluntary donation (blood bank)'
    WHERE abs(hashtext(u.email)) % 3 <> 0
) d
WHERE u.donated_at IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM public.donation_history x
    WHERE x.donor_id = u.id AND x.donation_date = d.donation_date
)
ORDER BY d.donation_date;


DROP TABLE IF EXISTS pg_temp.seed_users;
