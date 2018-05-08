package fuse

const ErrNoXattr = errNoXattr

var _ error = ErrNoXattr
var _ Errno = ErrNoXattr
var _ ErrorNumber = ErrNoXattr
