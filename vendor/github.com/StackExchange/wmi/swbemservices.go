// +build windows

package wmi

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type SWbemServices struct {

	cWMIClient            *Client 
	sWbemLocatorIUnknown  *ole.IUnknown
	sWbemLocatorIDispatch *ole.IDispatch
	queries               chan *queryRequest
	closeError            chan error
	lQueryorClose         sync.Mutex
}

type queryRequest struct {
	query    string
	dst      interface{}
	args     []interface{}
	finished chan error
}

func InitializeSWbemServices(c *Client, connectServerArgs ...interface{}) (*SWbemServices, error) {

	s := new(SWbemServices)
	s.cWMIClient = c
	s.queries = make(chan *queryRequest)
	initError := make(chan error)
	go s.process(initError)

	err, ok := <-initError
	if ok {
		return nil, err 
	}

	return s, nil
}

func (s *SWbemServices) Close() error {
	s.lQueryorClose.Lock()
	if s == nil || s.sWbemLocatorIDispatch == nil {
		s.lQueryorClose.Unlock()
		return fmt.Errorf("SWbemServices is not Initialized")
	}
	if s.queries == nil {
		s.lQueryorClose.Unlock()
		return fmt.Errorf("SWbemServices has been closed")
	}

	var result error
	ce := make(chan error)
	s.closeError = ce 
	close(s.queries)  
	s.lQueryorClose.Unlock()
	err, ok := <-ce
	if ok {
		result = err
	}

	return result
}

func (s *SWbemServices) process(initError chan error) {

	runtime.LockOSThread()
	defer runtime.LockOSThread()

	err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	if err != nil {
		oleCode := err.(*ole.OleError).Code()
		if oleCode != ole.S_OK && oleCode != S_FALSE {
			initError <- fmt.Errorf("ole.CoInitializeEx error: %v", err)
			return
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		initError <- fmt.Errorf("CreateObject SWbemLocator error: %v", err)
		return
	} else if unknown == nil {
		initError <- ErrNilCreateObject
		return
	}
	defer unknown.Release()
	s.sWbemLocatorIUnknown = unknown

	dispatch, err := s.sWbemLocatorIUnknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		initError <- fmt.Errorf("SWbemLocator QueryInterface error: %v", err)
		return
	}
	defer dispatch.Release()
	s.sWbemLocatorIDispatch = dispatch

	close(initError)

	for q := range s.queries {

		errQuery := s.queryBackground(q)

		if errQuery != nil {
			q.finished <- errQuery
		}
		close(q.finished)
	}

	s.queries = nil 

	close(s.closeError)
}

func (s *SWbemServices) Query(query string, dst interface{}, connectServerArgs ...interface{}) error {
	s.lQueryorClose.Lock()
	if s == nil || s.sWbemLocatorIDispatch == nil {
		s.lQueryorClose.Unlock()
		return fmt.Errorf("SWbemServices is not Initialized")
	}
	if s.queries == nil {
		s.lQueryorClose.Unlock()
		return fmt.Errorf("SWbemServices has been closed")
	}

	qr := queryRequest{
		query:    query,
		dst:      dst,
		args:     connectServerArgs,
		finished: make(chan error),
	}
	s.queries <- &qr
	s.lQueryorClose.Unlock()
	err, ok := <-qr.finished
	if ok {

		return err 
	}

	return nil
}

func (s *SWbemServices) queryBackground(q *queryRequest) error {
	if s == nil || s.sWbemLocatorIDispatch == nil {
		return fmt.Errorf("SWbemServices is not Initialized")
	}
	wmi := s.sWbemLocatorIDispatch 

	dv := reflect.ValueOf(q.dst)
	if dv.Kind() != reflect.Ptr || dv.IsNil() {
		return ErrInvalidEntityType
	}
	dv = dv.Elem()
	mat, elemType := checkMultiArg(dv)
	if mat == multiArgTypeInvalid {
		return ErrInvalidEntityType
	}

	serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", q.args...)
	if err != nil {
		return err
	}
	service := serviceRaw.ToIDispatch()
	defer serviceRaw.Clear()

	resultRaw, err := oleutil.CallMethod(service, "ExecQuery", q.query)
	if err != nil {
		return err
	}
	result := resultRaw.ToIDispatch()
	defer resultRaw.Clear()

	count, err := oleInt64(result, "Count")
	if err != nil {
		return err
	}

	enumProperty, err := result.GetProperty("_NewEnum")
	if err != nil {
		return err
	}
	defer enumProperty.Clear()

	enum, err := enumProperty.ToIUnknown().IEnumVARIANT(ole.IID_IEnumVariant)
	if err != nil {
		return err
	}
	if enum == nil {
		return fmt.Errorf("can't get IEnumVARIANT, enum is nil")
	}
	defer enum.Release()

	dv.Set(reflect.MakeSlice(dv.Type(), 0, int(count)))

	var errFieldMismatch error
	for itemRaw, length, err := enum.Next(1); length > 0; itemRaw, length, err = enum.Next(1) {
		if err != nil {
			return err
		}

		err := func() error {

			item := itemRaw.ToIDispatch()
			defer item.Release()

			ev := reflect.New(elemType)
			if err = s.cWMIClient.loadEntity(ev.Interface(), item); err != nil {
				if _, ok := err.(*ErrFieldMismatch); ok {

					errFieldMismatch = err
				} else {
					return err
				}
			}
			if mat != multiArgTypeStructPtr {
				ev = ev.Elem()
			}
			dv.Set(reflect.Append(dv, ev))
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return errFieldMismatch
}
