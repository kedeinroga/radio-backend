-- Fix country field length
ALTER TABLE stations ALTER COLUMN country TYPE VARCHAR(100);
