-- Users table
CREATE TABLE IF NOT EXISTS public.users (
    id serial PRIMARY KEY,
    firstname varchar(255),
    lastname varchar(255),
    gender varchar(50),
    password varchar(255),
    email varchar(255) UNIQUE,
    phone varchar(50),
    address text,
    city varchar(100),
    district varchar(100),
    division varchar(100),
    country varchar(100),
    is_available boolean DEFAULT true,
    isverified boolean DEFAULT false,
    lastlogin timestamp,
    donated_at timestamp,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    created_by integer,
    updated_by integer,
    profession varchar(100),
    is_password_temporary boolean DEFAULT false,
    id_role integer,
    verification_code varchar(255),
    hide_personal_info boolean DEFAULT false
);


-- Sessions table
CREATE TABLE IF NOT EXISTS public.sessions (
    id serial PRIMARY KEY,
    token text UNIQUE NOT NULL,
    role_id integer,
    user_id integer,
    loggedin_profile_id integer,
    is_valid boolean DEFAULT true,
    last_used timestamp,
    created_at timestamp DEFAULT now(),
    currency varchar(10),
    expire_at timestamp,
    iso_country varchar(10)
);

-- Role table
CREATE TABLE IF NOT EXISTS public.Role ( 
    id serial PRIMARY key, 
    name varchar(255) NOT NULL, 
    description varchar(255), 
    created_by integer, 
    updated_by integer, 
    created_at timestamp without time zone DEFAULT now(), 
    updated_at timestamp without time zone DEFAULT now(), 
    isdeleted bool default false 
); 

ALTER TABLE public.Role OWNER TO postgres; 

-- Permissions table
CREATE TABLE IF NOT EXISTS public.permission ( 
    id serial PRIMARY key, 
    name varchar(255) NOT NULL, 
    onlysuperadmin bool default false, 
    description text default '' 
); 

ALTER TABLE public.permission OWNER TO postgres; 

-- role_permission table
CREATE TABLE IF NOT EXISTS public.role_permission ( 
    id serial PRIMARY key, 
    role_id integer NOT NULL, 
    permission_id integer NOT NULL 
); 

ALTER TABLE public.role_permission OWNER TO postgres;

-- User Locations table
CREATE TABLE IF NOT EXISTS public.user_locations (
    id serial PRIMARY KEY,
    country varchar(100),
    country_geo_id integer,
    city varchar(100),
    city_geo_id integer,
    is_trusted boolean DEFAULT false,
    user_id integer,
    trust_code varchar(255),
    created_at timestamp DEFAULT now()
);

-- Insert default roles
INSERT INTO public.Role (id, name, description, isdeleted) VALUES
(1, 'SUPERADMIN', 'Super Administrator', false),
(2, 'ADMIN', 'Administrator', false),
(3, 'MANAGER', 'Manager', false),
(4, 'USERROLE', 'User', false)
ON CONFLICT (id) DO NOTHING;

-- Reset sequence for Role table
SELECT setval('public.role_id_seq', (SELECT MAX(id) FROM public.Role));

-- Insert default users (Superadmin & Admin)
-- Password is Admin@1234
INSERT INTO public.users (
    firstname, lastname, gender, email, password, phone, city, address, country, verification_code,
    id_role, is_available, isverified, created_at, updated_at
) VALUES 
('Super', 'Admin', 'M', 'superadmin@blood-link.online', '$2a$10$JN6X.THRe8nrKzpWeY4Xrupd3Y4tKvVEHrmF8IsjyZxgBn0pU9Ake', '', '', '', '', '',
 1, true, true, now(), now()),

('Admin', 'User', 'M', 'admin@blood-link.online', '$2a$10$JN6X.THRe8nrKzpWeY4Xrupd3Y4tKvVEHrmF8IsjyZxgBn0pU9Ake', '', '', '', '', '',
 2, true, true, now(), now())
ON CONFLICT (email) DO NOTHING;

-- Normalize legacy non-snake_case columns to snake_case.
-- Uses conditional renames for safer re-runs across environments.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'users'
          AND column_name = 'isverified'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'users'
          AND column_name = 'is_verified'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN isverified TO is_verified;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'users'
          AND column_name = 'lastlogin'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'users'
          AND column_name = 'last_login'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN lastlogin TO last_login;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'permission'
          AND column_name = 'onlysuperadmin'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'permission'
          AND column_name = 'only_superadmin'
    ) THEN
        ALTER TABLE public.permission RENAME COLUMN onlysuperadmin TO only_superadmin;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'role'
          AND column_name = 'isdeleted'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'role'
          AND column_name = 'is_deleted'
    ) THEN
        ALTER TABLE public.role RENAME COLUMN isdeleted TO is_deleted;
    END IF;
END $$;

-- Normalize legacy non-snake_case identifiers to snake_case across commonly used tables.
-- Uses conditional renames for safer re-runs across environments.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'firstname'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'first_name'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN firstname TO first_name;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'lastname'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'last_name'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN lastname TO last_name;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'id_role'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'role_id'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN id_role TO role_id;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'idrole'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'role_id'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN idrole TO role_id;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'verificationcode'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'verification_code'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN verificationcode TO verification_code;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'ispasswordtemp'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'is_password_temporary'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN ispasswordtemp TO is_password_temporary;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'hidepersonalinfo'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'hide_personal_info'
    ) THEN
        ALTER TABLE public.users RENAME COLUMN hidepersonalinfo TO hide_personal_info;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'sessions' AND column_name = 'lastused'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'sessions' AND column_name = 'last_used'
    ) THEN
        ALTER TABLE public.sessions RENAME COLUMN lastused TO last_used;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'sessions' AND column_name = 'expireat'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'sessions' AND column_name = 'expire_at'
    ) THEN
        ALTER TABLE public.sessions RENAME COLUMN expireat TO expire_at;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'sessions' AND column_name = 'isocountry'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'sessions' AND column_name = 'iso_country'
    ) THEN
        ALTER TABLE public.sessions RENAME COLUMN isocountry TO iso_country;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'representative' AND column_name = 'createdby'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'representative' AND column_name = 'created_by'
    ) THEN
        ALTER TABLE public.representative RENAME COLUMN createdby TO created_by;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'professional' AND column_name = 'createdby'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'professional' AND column_name = 'created_by'
    ) THEN
        ALTER TABLE public.professional RENAME COLUMN createdby TO created_by;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'representative' AND column_name = 'isdeleted'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'representative' AND column_name = 'is_deleted'
    ) THEN
        ALTER TABLE public.representative RENAME COLUMN isdeleted TO is_deleted;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'professional' AND column_name = 'isdeleted'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'professional' AND column_name = 'is_deleted'
    ) THEN
        ALTER TABLE public.professional RENAME COLUMN isdeleted TO is_deleted;
    END IF;
END $$;


