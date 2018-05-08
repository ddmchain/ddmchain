package oleutil

import ole "github.com/go-ole/go-ole"

func ClassIDFrom(programID string) (classID *ole.GUID, err error) {
	return ole.ClassIDFrom(programID)
}

func CreateObject(programID string) (unknown *ole.IUnknown, err error) {
	classID, err := ole.ClassIDFrom(programID)
	if err != nil {
		return
	}

	unknown, err = ole.CreateInstance(classID, ole.IID_IUnknown)
	if err != nil {
		return
	}

	return
}

func GetActiveObject(programID string) (unknown *ole.IUnknown, err error) {
	classID, err := ole.ClassIDFrom(programID)
	if err != nil {
		return
	}

	unknown, err = ole.GetActiveObject(classID, ole.IID_IUnknown)
	if err != nil {
		return
	}

	return
}

func CallMethod(disp *ole.IDispatch, name string, params ...interface{}) (result *ole.VARIANT, err error) {
	return disp.InvokeWithOptionalArgs(name, ole.DISPATCH_METHOD, params)
}

func MustCallMethod(disp *ole.IDispatch, name string, params ...interface{}) (result *ole.VARIANT) {
	r, err := CallMethod(disp, name, params...)
	if err != nil {
		panic(err.Error())
	}
	return r
}

func GetProperty(disp *ole.IDispatch, name string, params ...interface{}) (result *ole.VARIANT, err error) {
	return disp.InvokeWithOptionalArgs(name, ole.DISPATCH_PROPERTYGET, params)
}

func MustGetProperty(disp *ole.IDispatch, name string, params ...interface{}) (result *ole.VARIANT) {
	r, err := GetProperty(disp, name, params...)
	if err != nil {
		panic(err.Error())
	}
	return r
}

func PutProperty(disp *ole.IDispatch, name string, params ...interface{}) (result *ole.VARIANT, err error) {
	return disp.InvokeWithOptionalArgs(name, ole.DISPATCH_PROPERTYPUT, params)
}

func MustPutProperty(disp *ole.IDispatch, name string, params ...interface{}) (result *ole.VARIANT) {
	r, err := PutProperty(disp, name, params...)
	if err != nil {
		panic(err.Error())
	}
	return r
}

func PutPropertyRef(disp *ole.IDispatch, name string, params ...interface{}) (result *ole.VARIANT, err error) {
	return disp.InvokeWithOptionalArgs(name, ole.DISPATCH_PROPERTYPUTREF, params)
}

func MustPutPropertyRef(disp *ole.IDispatch, name string, params ...interface{}) (result *ole.VARIANT) {
	r, err := PutPropertyRef(disp, name, params...)
	if err != nil {
		panic(err.Error())
	}
	return r
}

func ForEach(disp *ole.IDispatch, f func(v *ole.VARIANT) error) error {
	newEnum, err := disp.GetProperty("_NewEnum")
	if err != nil {
		return err
	}
	defer newEnum.Clear()

	enum, err := newEnum.ToIUnknown().IEnumVARIANT(ole.IID_IEnumVariant)
	if err != nil {
		return err
	}
	defer enum.Release()

	for item, length, err := enum.Next(1); length > 0; item, length, err = enum.Next(1) {
		if err != nil {
			return err
		}
		if ferr := f(&item); ferr != nil {
			return ferr
		}
	}
	return nil
}
