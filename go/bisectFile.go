/*
  Binary search of text file with variable length lines.
  for first keys equal to or greather than desired key.
  
  Since variable length lines mean we can and normally
  will land in the middle of a line we do a readln() to
  get to end of current line and another readln() to get
  the next full line for comparison. 
  
  Binary search has some nodes such as middle of file 
  that are visted quite often. We cache the strings
  at those positions to avoid the seek and read when
  we have already visited those points.  We keep a limited
  set of these to control memory usage so we keep a counter
  of times visited for each node.  When the list reaches 
  the max size we periodically process it it and throw 
  away the least recently used 30%. 
  
  To make things easier for next segment we provide 
  function to return all lines that match the specified
  suffix
*/

