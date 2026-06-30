
- In proto files how does Service and rpc combination work?

- For persisting data, we will use badgerDB. A key value store with txn support. We would only store the docId -> docContent mapping and not the inverted index. Inverted index to be build from scratch on service start.

- Check time taken to rebuilt index.

- For  every insert we add to badgerDB. What will be the flushing strategy? Do we flush after every insert or do we batch them and flush after a certain threshold?

- good reference code for inmemory table of skiplist and wal | trie in badger db code.  


## Step 8: Query router fan-out for search queries

- router gets the search request from gateway. And then does a search on indexnodes.
- Suppose we have 3 index nodes then router calls each one in parallel. Thus we can keep latency budget here like 30ms for complete request.
- using `context.WithTimeout`per shard call to avoid one shard holding the complete request.
- reuse grpc clients/connections
- we are ok with returning partial results in case one shard doesn't respond.
- gateway calls router through grpc
- In router search: we fetch all active nodes from etcd watcher, do a fanout to nodes and then merge results from shard and then sort by score. `That is a good production habit. Keep orchestration easy to scan, push detailed concurrency into a helper.`
- 
