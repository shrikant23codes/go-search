package index

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

//  The closure pattern is deliberate: when the function returns, Badger
//   commits (Update) or releases (View) the transaction. If your closure
//   returns an error, Update rolls back. No manual Begin/Commit/Rollback like
//    SQL. Cleaner, harder to forget.

// In badgerDB there are namespaces for document keys(docPrefix).
// FUll key = docPrefix + key
const docPrefix = "doc:"

// On open we load from disk and reindex the inmemory Index

func Open(path string) (*Index, error) {
	opts := badger.DefaultOptions(path).WithLogger(nil)
	db, err := badger.Open(opts)

	if err != nil {
		return nil, fmt.Errorf("failed to open badgerDB: %w", err)
	}

	idx := &Index{
		postings:  make(map[string]*PostingsList),
		docs:      make(map[uint64]Document),
		docLens:   make(map[uint64]int),
		tokenizer: NewStandardTokenizer(),
		db:        db,
	}

	if err := idx.LoadFromDB(); err != nil {
		return nil, fmt.Errorf("failed to load index from badgerDB: %w", err)
	}

	return idx, nil
}

func (idx *Index) Close() error {
	if idx.db == nil {
		return nil
	}
	return idx.db.Close()
}

func (idx *Index) persistDoc(doc Document) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document %q: %w", doc.ID, err)
	}

	return idx.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(docPrefix+doc.ID), data)
	})
}

// traverses keys in lexicographic (sorted) order because of LSM tree structure.
func (idx *Index) LoadFromDB() error {
	return idx.db.View(func(txn *badger.Txn) error {
		// There is a key only iterator also available.
		// We can have multiple read iterators.
		// For read-write iterator it works on snapshot of data at the time of txn creation, so it won't see any changes after that.
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		// iterator hold resources.. close them.
		defer it.Close()

		prefix := []byte(docPrefix)
		// We do scan on prefix
		// This is the std pattern for any LSM-style KV store: prefix scans replace SQL WHERE key LIKE 'doc:%'
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			var doc Document
			err := it.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &doc)
			})

			if err != nil {
				return fmt.Errorf("failed to unmarshal document for key %s: %w", it.Item().Key(), err)
			}

			if err := idx.addInMemory(doc); err != nil {
				return fmt.Errorf("failed to add document %q to in-memory index: %w", doc.ID, err)
			}
		}
		return nil
	})
}
