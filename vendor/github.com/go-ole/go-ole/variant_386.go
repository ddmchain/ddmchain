// +build 386

package ole

type VARIANT struct {
	VT         VT     
	wReserved1 uint16 
	wReserved2 uint16 
	wReserved3 uint16 
	Val        int64  
}
