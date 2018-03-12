import redis
import sys 

rhost = sys.argv[1]
port = 6379
db = 0

r = redis.StrictRedis(host=rhost, port=port, db=db)

answer = raw_input("Are you sure you want to delete all data in the database %s:%s which contains %d keys (y/n)?" % (rhost,port,len(r.keys())))
if answer.lower() == 'y':
    r.flushall()
    print("Purged database successfully")
else:
    print("exiting without purge")