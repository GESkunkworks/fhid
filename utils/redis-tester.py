import redis
import sys 

rhost = sys.argv[1]

r = redis.StrictRedis(host=rhost, port=6379, db=0)

#for key in r.scan_iter():
#    print key 

print r.get(sys.argv[2])
