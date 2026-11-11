-- =====================================================================
-- 000002_seed_dummy_data (DOWN)
-- Reverts everything 000002 added and leaves the database as it was
-- right after 000001.
--
-- WARNING: dropping the donation_* tables also removes any real
-- requests, posts or history created after this migration ran.
-- =====================================================================


-- 1. Donation tables (children first). This also removes every seeded
--    blood request, including the test-scenario requests made by
--    superadmin, admin and donor@blood-link.online.
DROP TABLE IF EXISTS public.donation_history;
DROP TABLE IF EXISTS public.donation_post_interests;
DROP TABLE IF EXISTS public.donation_posts;
DROP TABLE IF EXISTS public.donation_requests;


-- 2. Seeded users plus their sessions and locations
CREATE TEMP TABLE seed_emails (email varchar(255) PRIMARY KEY);
INSERT INTO seed_emails (email) VALUES
('donor@blood-link.online'),
('kamrul.hasan@blood-link.online'), ('shamima.nasrin@blood-link.online'), ('abu.sayeed@blood-link.online'),
('tanvir.ahmed92@gmail.com'), ('nusrat.jahan@yahoo.com'), ('mahmud.hasan.du@gmail.com'),
('sadia.islam07@gmail.com'), ('rafiqul.islam.savar@gmail.com'), ('farhana.akter88@gmail.com'),
('arif.hossain.bd@outlook.com'), ('sumaiya.rahman@gmail.com'), ('shakil.ahmed.ng@gmail.com'),
('ayesha.siddika@gmail.com'), ('imran.kabir.mpur@gmail.com'), ('jannatul.ferdous99@gmail.com'),
('sabbir.rahman@gmail.com'), ('tahmina.akhter@yahoo.com'), ('nazmul.huda.fp@gmail.com'),
('mehedi.hasan.bhairab@gmail.com'), ('rumana.afroz@gmail.com'), ('asif.iqbal.gulshan@gmail.com'),
('shirin.sultana@gmail.com'), ('rakibul.hasan.ju@gmail.com'), ('sanjida.haque@gmail.com'),
('saiful.alam.ctg@gmail.com'), ('tasnim.chowdhury@gmail.com'), ('jahidul.islam.ctg@gmail.com'),
('nadia.sultana.cox@gmail.com'), ('monirul.islam.cumilla@gmail.com'), ('shahana.parvin@gmail.com'),
('ariful.haque.noakhali@gmail.com'), ('rehana.begum.bb@gmail.com'), ('fahim.uddin.cuet@gmail.com'),
('mithila.das@gmail.com'), ('kawsar.ahmed.chandpur@gmail.com'), ('habibur.rahman.rj@gmail.com'),
('sharmin.akter.ru@gmail.com'), ('mostafiz.bogura@gmail.com'), ('lipi.khatun@gmail.com'),
('almamun.sirajganj@gmail.com'), ('rubel.hossain.natore@gmail.com'), ('sohel.rana.khulna@gmail.com'),
('moumita.saha@gmail.com'), ('ashraful.alam.kushtia@gmail.com'), ('rokeya.khanam@gmail.com'),
('biplob.ghosh@gmail.com'), ('rezaul.karim.syl@gmail.com'), ('fatema.zohra@gmail.com'),
('jubayer.ahmed.sust@gmail.com'), ('anwar.hossain.srimangal@gmail.com'), ('mizanur.rahman.rangpur@gmail.com'),
('shapla.begum@gmail.com'), ('nayeem.islam.saidpur@gmail.com'), ('kamal.uddin.bsl@gmail.com'),
('taslima.akter.bsl@gmail.com'), ('hasan.mahmud.bhola@gmail.com'), ('sharif.hossain.ptk@gmail.com'),
('rashed.khan.mym@gmail.com'), ('mousumi.akter.bau@gmail.com'), ('rafsan.jani@gmail.com');

DELETE FROM public.sessions
WHERE user_id IN (SELECT u.id FROM public.users u JOIN seed_emails s ON s.email = u.email);

DELETE FROM public.user_locations
WHERE user_id IN (SELECT u.id FROM public.users u JOIN seed_emails s ON s.email = u.email);

DELETE FROM public.users
WHERE email IN (SELECT email FROM seed_emails);

DROP TABLE seed_emails;


-- 3. Role ↔ permission mapping and permissions
DELETE FROM public.role_permission rp
USING (VALUES
    (1, 1), (1, 2), (1, 3), (1, 4), (1, 5), (1, 6), (1, 7), (1, 8), (1, 9),
    (2, 1), (2, 3), (2, 4), (2, 5), (2, 6), (2, 8), (2, 9),
    (3, 1), (3, 4), (3, 5), (3, 6), (3, 8), (3, 9),
    (4, 1), (4, 4), (4, 5), (4, 6), (4, 9)
) AS v(role_id, permission_id)
WHERE rp.role_id = v.role_id AND rp.permission_id = v.permission_id;

DELETE FROM public.permission p
USING (VALUES
    (1, 'CHANGE_PASSWORD'), (2, 'RESET_PASSWORD'), (3, 'CREATE_USER'),
    (4, 'UPDATE_USER'), (5, 'LIST_USER'), (6, 'VIEW_USER'),
    (7, 'DELETE_USER'), (8, 'TOGGLE_USER_STATUS'), (9, 'PROFILE_LOGIN')
) AS v(id, name)
WHERE p.id = v.id AND p.name = v.name;

SELECT setval('public.permission_id_seq',
              COALESCE((SELECT MAX(id) FROM public.permission), 1),
              (SELECT MAX(id) FROM public.permission) IS NOT NULL);
SELECT setval('public.role_permission_id_seq',
              COALESCE((SELECT MAX(id) FROM public.role_permission), 1),
              (SELECT MAX(id) FROM public.role_permission) IS NOT NULL);


-- 4. Default admins back to their 000001 values
UPDATE public.users
SET country = '', city = '', district = NULL, division = NULL
WHERE email IN ('superadmin@blood-link.online', 'admin@blood-link.online')
  AND country = 'Bangladesh' AND city = 'Dhaka';


-- 5. Columns added to users
ALTER TABLE public.users DROP COLUMN IF EXISTS photo;
ALTER TABLE public.users DROP COLUMN IF EXISTS date_of_birth;
ALTER TABLE public.users DROP COLUMN IF EXISTS blood_group;


-- 6. Rewind the users / sessions / user_locations sequences
SELECT setval('public.users_id_seq',
              COALESCE((SELECT MAX(id) FROM public.users), 1),
              (SELECT MAX(id) FROM public.users) IS NOT NULL);
SELECT setval('public.sessions_id_seq',
              COALESCE((SELECT MAX(id) FROM public.sessions), 1),
              (SELECT MAX(id) FROM public.sessions) IS NOT NULL);
SELECT setval('public.user_locations_id_seq',
              COALESCE((SELECT MAX(id) FROM public.user_locations), 1),
              (SELECT MAX(id) FROM public.user_locations) IS NOT NULL);
