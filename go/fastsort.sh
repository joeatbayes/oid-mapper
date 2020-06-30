time rm ../data/index/*.seg
time ./bisMakeSeg ../data/stage/generated_oids.340m.map.txt ../data/index/340m > t.t1
time ./bisMergeSegMultBulk ../data/index/340m /home/jwork/index/340merged /home/jwork/index/340-000-index.seg > t.t2
