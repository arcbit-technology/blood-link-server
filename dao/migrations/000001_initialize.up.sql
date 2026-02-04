-- Users table
CREATE TABLE IF NOT EXISTS public.users (
    id serial PRIMARY KEY,
    firstname varchar(255),
    lastname varchar(255),
    password varchar(255),
    username varchar(255),
    email varchar(255) UNIQUE,
    phone varchar(50),
    address text,
    city varchar(100),
    district varchar(100),
    division varchar(100),
    country varchar(100),
    isactive boolean DEFAULT true,
    isverified boolean DEFAULT false,
    lastlogin timestamp,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    created_by integer,
    updated_by integer,
    profession varchar(100),
    is_password_temporary boolean DEFAULT false,
    id_role integer,
    verification_code varchar(255)
);


-- Sessions table
CREATE TABLE IF NOT EXISTS public.sessions (
    id serial PRIMARY KEY,
    token text UNIQUE NOT NULL,
    role_id integer,
    user_id integer,
    buyer_profile_id integer,
    seller_profile_id integer,
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
    firstname, lastname, username, email, password, phone, city, address, country, verification_code,
    id_role, isactive, isverified, created_at, updated_at
) VALUES 
('Super', 'Admin', 'superadmin', 'superadmin@example.com', '$2a$10$JN6X.THRe8nrKzpWeY4Xrupd3Y4tKvVEHrmF8IsjyZxgBn0pU9Ake', '', '', '', '', '',
 1, true, true, now(), now()),

('Admin', 'User', 'admin', 'admin@example.com', '$2a$10$JN6X.THRe8nrKzpWeY4Xrupd3Y4tKvVEHrmF8IsjyZxgBn0pU9Ake', '', '', '', '', '',
 2, true, true, now(), now())
ON CONFLICT (email) DO NOTHING;


