
package ddm

import (
	"bytes"
	"time"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/ddmpv"
	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/ptl"
)

var deduplicateData = []byte("dbUpgrade_20170714deduplicateData")

func upgradeDeduplicateData(db ddmdb.Database) func() error {

	data, _ := db.Get(deduplicateData)
	if len(data) > 0 && data[0] == 42 {
		return nil
	}
	if data, _ := db.Get([]byte("LastHeader")); len(data) == 0 {
		db.Put(deduplicateData, []byte{42})
		return nil
	}

	log.Warn("Upgrading database to use lookup entries")
	stop := make(chan chan error)

	go func() {

		it := db.(*ddmdb.LDBDatabase).NewIterator()
		defer func() {
			if it != nil {
				it.Release()
			}
		}()

		var (
			converted uint64
			failed    error
		)
		for failed == nil && it.Next() {

			key := it.Key()
			if len(key) != common.HashLength+1 || key[common.HashLength] != 0x01 {
				continue
			}

			var meta struct {
				BlockHash  common.Hash
				BlockIndex uint64
				Index      uint64
			}
			if err := rlp.DecodeBytes(it.Value(), &meta); err != nil {
				continue
			}

			hash := key[:common.HashLength]

			if hash[0] == byte('l') {

				if tx, _, _, _ := core.GetTransaction(db, common.BytesToHash(hash)); tx == nil || !bytes.Equal(tx.Hash().Bytes(), hash) {
					continue
				}
			}

			if failed = db.Put(append([]byte("l"), hash...), it.Value()); failed == nil { 
				if failed = db.Delete(hash); failed == nil { 
					if failed = db.Delete(append([]byte("receipts-"), hash...)); failed == nil { 
						if failed = db.Delete(key); failed != nil { 
							break
						}
					}
				}
			}

			converted++
			if converted%100000 == 0 {
				it.Release()
				it = db.(*ddmdb.LDBDatabase).NewIterator()
				it.Seek(key)

				log.Info("Deduplicating database entries", "deduped", converted)
			}

			select {
			case errc := <-stop:
				errc <- nil
				return
			case <-time.After(time.Microsecond * 100):
			}
		}

		if failed == nil {
			log.Info("Database deduplication successful", "deduped", converted)
			db.Put(deduplicateData, []byte{42})
		} else {
			log.Error("Database deduplication failed", "deduped", converted, "err", failed)
		}
		it.Release()
		it = nil

		errc := <-stop
		errc <- failed
	}()

	return func() error {
		errc := make(chan error)
		stop <- errc
		return <-errc
	}
}
