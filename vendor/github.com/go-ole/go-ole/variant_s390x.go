// +build s390x

package ole

type VARIANT struct {
	VT         VT      
	wReserved1 uint16  
	wReserved2 uint16  
	wReserved3 uint16  
	Val        int64   
	_          [8]byte 
}
