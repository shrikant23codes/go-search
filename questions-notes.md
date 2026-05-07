
- In proto files how does Service and rpc combination work?

- For persisting data, we will use badgerDB. A key value store with txn support. We would only store the docId -> docContent mapping and not the inverted index. Inverted index to be build from scratch on service start.

- Check time taken to rebuilt index.

- For  every insert we add to badgerDB. What will be the flushing strategy? Do we flush after every insert or do we batch them and flush after a certain threshold?

- good reference code for inmemory table of skiplist and wal | trie in badger db code.  