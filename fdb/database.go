// FoundationDB Go API
// Copyright (c) 2013 FoundationDB, LLC

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package fdb

/*
 #define FDB_API_VERSION 100
 #include <foundationdb/fdb_c.h>
*/
import "C"

import (
	"runtime"
)

type Database struct {
	d *C.FDBDatabase
	Options databaseOptions
}

type databaseOptions struct {
	database *Database
}

func (opt databaseOptions) setOpt(code int, param []byte) error {
	if opt.database == nil {
		return &Error{errorClientInvalidOperation}
	}

	return setOpt(func(p *C.uint8_t, pl C.int) C.fdb_error_t {
		return C.fdb_database_set_option(opt.database.d, C.FDBDatabaseOption(code), p, pl)
	}, param)
}

func (d *Database) destroy() {
	C.fdb_database_destroy(d.d)
}

func (d *Database) CreateTransaction() (*Transaction, error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	var outt *C.FDBTransaction

	if err := C.fdb_database_create_transaction(d.d, &outt); err != 0 {
		return nil, &Error{err}
	}

	t := &Transaction{t: outt}
	t.Options.transaction = t

	runtime.SetFinalizer(t, (*Transaction).destroy)

	return t, nil
}

func (d *Database) Transact(f func(tr *Transaction) (interface{}, error)) (ret interface{}, e error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	tr, e := d.CreateTransaction()
	/* Any error here is non-retryable */
	if e != nil {
		return
	}

	wrapped := func() {
		defer func() {
			if r := recover(); r != nil {
				switch r := r.(type) {
				case *Error:
					e = r
				default:
					panic(r)
				}
			}
		}()

		ret, e = f(tr)

		if e != nil {
			return
		}

		e = tr.Commit().GetWithError()
	}

	for {
		wrapped()

		/* No error means success! */
		if e == nil {
			return
		}

		switch ep := e.(type) {
		case *Error:
			e = tr.OnError(ep).GetWithError()
		}

		/* If OnError returns an error, then it's not
		/* retryable; otherwise take another pass at things */
		if e != nil {
			return
		}
	}
}

func (d *Database) Get(key []byte) ([]byte, error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	v, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		return tr.Get(key).GetOrPanic(), nil
	})
	if e != nil {
		return nil, e
	}
	return v.([]byte), nil
}

func (d *Database) GetKey(sel KeySelector) ([]byte, error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	v, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		return tr.GetKey(sel).GetOrPanic(), nil
	})
	if e != nil {
		return nil, e
	}
	return v.([]byte), nil
}

func (d *Database) GetRange(begin []byte, end []byte, options RangeOptions) ([]KeyValue, error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	v, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		return tr.GetRange(begin, end, options).GetSliceOrPanic(), nil
	})
	if e != nil {
		return nil, e
	}
	return v.([]KeyValue), nil
}

func (d *Database) GetRangeSelector(begin KeySelector, end KeySelector, options RangeOptions) ([]KeyValue, error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	v, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		return tr.GetRangeSelector(begin, end, options).GetSliceOrPanic(), nil
	})
	if e != nil {
		return nil, e
	}
	return v.([]KeyValue), nil
}

func (d *Database) GetRangeStartsWith(prefix []byte, options RangeOptions) ([]KeyValue, error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	v, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		return tr.GetRangeStartsWith(prefix, options).GetSliceOrPanic(), nil
	})
	if e != nil {
		return nil, e
	}
	return v.([]KeyValue), nil
}

func (d *Database) Set(key []byte, value []byte) error {
	if d.d == nil {
		return &Error{errorClientInvalidOperation}
	}

	_, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		tr.Set(key, value)
		return nil, nil
	})
	if e != nil {
		return e
	}
	return nil
}

func (d *Database) Clear(key []byte) error {
	if d.d == nil {
		return &Error{errorClientInvalidOperation}
	}

	_, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		tr.Clear(key)
		return nil, nil
	})
	if e != nil {
		return e
	}
	return nil
}

func (d *Database) ClearRange(begin []byte, end []byte) error {
	if d.d == nil {
		return &Error{errorClientInvalidOperation}
	}

	_, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		tr.ClearRange(begin, end)
		return nil, nil
	})
	if e != nil {
		return e
	}
	return nil
}

func (d *Database) ClearRangeStartsWith(prefix []byte) error {
	if d.d == nil {
		return &Error{errorClientInvalidOperation}
	}

	_, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		tr.ClearRangeStartsWith(prefix)
		return nil, nil
	})
	if e != nil {
		return e
	}
	return nil
}

func (d *Database) GetAndWatch(key []byte) ([]byte, *FutureNil, error) {
	if d.d == nil {
		return nil, nil, &Error{errorClientInvalidOperation}
	}

	r, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		v := tr.Get(key).GetOrPanic()
		w := tr.Watch(key)
		return struct{value []byte; watch *FutureNil}{v, w}, nil
		return nil, nil
	})
	if e != nil {
		return nil, nil, e
	}
	ret := r.(struct{value []byte; watch *FutureNil})
	return ret.value, ret.watch, nil
}

func (d *Database) SetAndWatch(key []byte, value []byte) (*FutureNil, error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	r, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		tr.Set(key, value)
		return tr.Watch(key), nil
	})
	if e != nil {
		return nil, e
	}
	return r.(*FutureNil), nil
}

func (d *Database) ClearAndWatch(key []byte) (*FutureNil, error) {
	if d.d == nil {
		return nil, &Error{errorClientInvalidOperation}
	}

	r, e := d.Transact(func (tr *Transaction) (interface{}, error) {
		tr.Clear(key)
		return tr.Watch(key), nil
	})
	if e != nil {
		return nil, e
	}
	return r.(*FutureNil), nil
}
