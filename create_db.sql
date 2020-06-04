-- Create Schema, Database & Table
CREATE DATABASE oidmap;

--SELECT DATABASE oidmap;
\c oidmap
DROP INDEX ondx;
DROP TABLE omap;
CREATE TABLE omap (
  partbl TEXT NOT NULL,
  paroid TEXT NOT NULL,
  chiltbl TEXT NOT NULL,
  chiloid TEXT NOT NULL
  
);

-- TODO: Test primary key in liue of the secondary index but
-- primary keys must be unqiue so we would need to add all 4
-- columns to it. 
-- PRIMARY KEY (chiloid, chiltbl, paroid, partbl)

CREATE INDEX CONCURRENTLY ondx ON omap ( chiloid, chiltbl)
