data/in includes directories to be processed 
and used to update the database
directories are created using a formulat cc/yy/mm/dayInMonth + dayInMont 
to prevent too many input files from accumulating in a single directory
files in these directories should be named as hhmmss when written and will
be processed in order.  If multiple files may be written during a single
second they must include a zero padded counter to allow a simple sort to
place them in sequence. 
