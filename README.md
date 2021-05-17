group 0,1,2



command
put key val
get key val
info
discon 0|1|2(group) 0|1|2(num)
con 0|1|2 0|1|2
leave 0|1|2
join 0|1|2

get /GetBasicInfo
{"MasterNum":3,"GroupNum":3,"Num":3,"IsMasterLeader":{"0":false,"1":true,"2":false},"IsLeader":{"0":{"0":false,"1":true,"2":false},"1":{"0":true,"1":false,"2":false},"2":{"0":false,"1":true,"2":false}},"IsMasterConnect":{"0":true,"1":true,"2":true},"IsConnect":{"0":{"0":true,"1":true,"2":true},"1":{"0":true,"1":true,"2":true},"2":{"0":true,"1":true,"2":true}},"MasterTerm":{"0":2,"1":2,"2":2},"Term":{"0":{"0":3,"1":3,"2":3},"1":{"0":2,"1":2,"2":2},"2":{"0":2,"1":2,"2":2}}}

post /GetMasterInfo
req
{
    "n":0
}
resp
{"Config":{"Num":1,"Shards":[100,100,100,100,101,101,101,102,102,102],"Groups":{"100":["server-100-0","server-100-1","server-100-2"],"101":["server-101-0","server-101-1","server-101-2"],"102":["server-102-0","server-102-1","server-102-2"]}}}


post /GetEntityInfo
req
{
    "n":0,
    "g":2
}
resp
{"ShardNum":4,"KeyNum":0,"IndexLen":2}


for i range 10000000:
    val = []
    for j in rafts:
        if j is leader:
            j.start(i)
            val.append(i)
    random rafts.disconnect 
    random tafts.sleep
    random rafts.restart
compare val rafts.log

