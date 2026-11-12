-- Cover image of a blood bank (Cloudinary URL). blood_banks.logo and
-- users.photo already exist.
ALTER TABLE public.blood_banks ADD COLUMN IF NOT EXISTS cover varchar(255);
