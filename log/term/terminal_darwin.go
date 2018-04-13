// Based on ssh/terminal:

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// +build !appengine

package term

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA

type Termios syscall.Termios
