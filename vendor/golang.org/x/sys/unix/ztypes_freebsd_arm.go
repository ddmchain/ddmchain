
// +build arm,freebsd

package unix

const (
	sizeofPtr      = 0x4
	sizeofShort    = 0x2
	sizeofInt      = 0x4
	sizeofLong     = 0x4
	sizeofLongLong = 0x8
)

type (
	_C_short     int16
	_C_int       int32
	_C_long      int32
	_C_long_long int64
)

type Timespec struct {
	Sec       int64
	Nsec      int32
	Pad_cgo_0 [4]byte
}

type Timeval struct {
	Sec       int64
	Usec      int32
	Pad_cgo_0 [4]byte
}

type Rusage struct {
	Utime    Timeval
	Stime    Timeval
	Maxrss   int32
	Ixrss    int32
	Idrss    int32
	Isrss    int32
	Minflt   int32
	Majflt   int32
	Nswap    int32
	Inblock  int32
	Oublock  int32
	Msgsnd   int32
	Msgrcv   int32
	Nsignals int32
	Nvcsw    int32
	Nivcsw   int32
}

type Rlimit struct {
	Cur int64
	Max int64
}

type _Gid_t uint32

const (
	S_IFMT   = 0xf000
	S_IFIFO  = 0x1000
	S_IFCHR  = 0x2000
	S_IFDIR  = 0x4000
	S_IFBLK  = 0x6000
	S_IFREG  = 0x8000
	S_IFLNK  = 0xa000
	S_IFSOCK = 0xc000
	S_ISUID  = 0x800
	S_ISGID  = 0x400
	S_ISVTX  = 0x200
	S_IRUSR  = 0x100
	S_IWUSR  = 0x80
	S_IXUSR  = 0x40
)

type Stat_t struct {
	Dev           uint32
	Ino           uint32
	Mode          uint16
	Nlink         uint16
	Uid           uint32
	Gid           uint32
	Rdev          uint32
	Atimespec     Timespec
	Mtimespec     Timespec
	Ctimespec     Timespec
	Size          int64
	Blocks        int64
	Blksize       int32
	Flags         uint32
	Gen           uint32
	Lspare        int32
	Birthtimespec Timespec
}

type Statfs_t struct {
	Version     uint32
	Type        uint32
	Flags       uint64
	Bsize       uint64
	Iosize      uint64
	Blocks      uint64
	Bfree       uint64
	Bavail      int64
	Files       uint64
	Ffree       int64
	Syncwrites  uint64
	Asyncwrites uint64
	Syncreads   uint64
	Asyncreads  uint64
	Spare       [10]uint64
	Namemax     uint32
	Owner       uint32
	Fsid        Fsid
	Charspare   [80]int8
	Fstypename  [16]int8
	Mntfromname [88]int8
	Mntonname   [88]int8
}

type Flock_t struct {
	Start     int64
	Len       int64
	Pid       int32
	Type      int16
	Whence    int16
	Sysid     int32
	Pad_cgo_0 [4]byte
}

type Dirent struct {
	Fileno uint32
	Reclen uint16
	Type   uint8
	Namlen uint8
	Name   [256]int8
}

type Fsid struct {
	Val [2]int32
}

const (
	FADV_NORMAL     = 0x0
	FADV_RANDOM     = 0x1
	FADV_SEQUENTIAL = 0x2
	FADV_WILLNEED   = 0x3
	FADV_DONTNEED   = 0x4
	FADV_NOREUSE    = 0x5
)

type RawSockaddrInet4 struct {
	Len    uint8
	Family uint8
	Port   uint16
	Addr   [4]byte 
	Zero   [8]int8
}

type RawSockaddrInet6 struct {
	Len      uint8
	Family   uint8
	Port     uint16
	Flowinfo uint32
	Addr     [16]byte 
	Scope_id uint32
}

type RawSockaddrUnix struct {
	Len    uint8
	Family uint8
	Path   [104]int8
}

type RawSockaddrDatalink struct {
	Len    uint8
	Family uint8
	Index  uint16
	Type   uint8
	Nlen   uint8
	Alen   uint8
	Slen   uint8
	Data   [46]int8
}

type RawSockaddr struct {
	Len    uint8
	Family uint8
	Data   [14]int8
}

type RawSockaddrAny struct {
	Addr RawSockaddr
	Pad  [92]int8
}

type _Socklen uint32

type Linger struct {
	Onoff  int32
	Linger int32
}

type Iovec struct {
	Base *byte
	Len  uint32
}

type IPMreq struct {
	Multiaddr [4]byte 
	Interface [4]byte 
}

type IPMreqn struct {
	Multiaddr [4]byte 
	Address   [4]byte 
	Ifindex   int32
}

type IPv6Mreq struct {
	Multiaddr [16]byte 
	Interface uint32
}

type Msghdr struct {
	Name       *byte
	Namelen    uint32
	Iov        *Iovec
	Iovlen     int32
	Control    *byte
	Controllen uint32
	Flags      int32
}

type Cmsghdr struct {
	Len   uint32
	Level int32
	Type  int32
}

type Inet6Pktinfo struct {
	Addr    [16]byte 
	Ifindex uint32
}

type IPv6MTUInfo struct {
	Addr RawSockaddrInet6
	Mtu  uint32
}

type ICMPv6Filter struct {
	Filt [8]uint32
}

const (
	SizeofSockaddrInet4    = 0x10
	SizeofSockaddrInet6    = 0x1c
	SizeofSockaddrAny      = 0x6c
	SizeofSockaddrUnix     = 0x6a
	SizeofSockaddrDatalink = 0x36
	SizeofLinger           = 0x8
	SizeofIPMreq           = 0x8
	SizeofIPMreqn          = 0xc
	SizeofIPv6Mreq         = 0x14
	SizeofMsghdr           = 0x1c
	SizeofCmsghdr          = 0xc
	SizeofInet6Pktinfo     = 0x14
	SizeofIPv6MTUInfo      = 0x20
	SizeofICMPv6Filter     = 0x20
)

const (
	PTRACE_TRACEME = 0x0
	PTRACE_CONT    = 0x7
	PTRACE_KILL    = 0x8
)

type Kevent_t struct {
	Ident  uint32
	Filter int16
	Flags  uint16
	Fflags uint32
	Data   int32
	Udata  *byte
}

type FdSet struct {
	X__fds_bits [32]uint32
}

const (
	sizeofIfMsghdr         = 0xa8
	SizeofIfMsghdr         = 0x70
	sizeofIfData           = 0x98
	SizeofIfData           = 0x60
	SizeofIfaMsghdr        = 0x14
	SizeofIfmaMsghdr       = 0x10
	SizeofIfAnnounceMsghdr = 0x18
	SizeofRtMsghdr         = 0x5c
	SizeofRtMetrics        = 0x38
)

type ifMsghdr struct {
	Msglen    uint16
	Version   uint8
	Type      uint8
	Addrs     int32
	Flags     int32
	Index     uint16
	Pad_cgo_0 [2]byte
	Data      ifData
}

type IfMsghdr struct {
	Msglen    uint16
	Version   uint8
	Type      uint8
	Addrs     int32
	Flags     int32
	Index     uint16
	Pad_cgo_0 [2]byte
	Data      IfData
}

type ifData struct {
	Type              uint8
	Physical          uint8
	Addrlen           uint8
	Hdrlen            uint8
	Link_state        uint8
	Vhid              uint8
	Datalen           uint16
	Mtu               uint32
	Metric            uint32
	Baudrate          uint64
	Ipackets          uint64
	Ierrors           uint64
	Opackets          uint64
	Oerrors           uint64
	Collisions        uint64
	Ibytes            uint64
	Obytes            uint64
	Imcasts           uint64
	Omcasts           uint64
	Iqdrops           uint64
	Oqdrops           uint64
	Noproto           uint64
	Hwassist          uint64
	X__ifi_epoch      [8]byte
	X__ifi_lastchange [16]byte
}

type IfData struct {
	Type        uint8
	Physical    uint8
	Addrlen     uint8
	Hdrlen      uint8
	Link_state  uint8
	Spare_char1 uint8
	Spare_char2 uint8
	Datalen     uint8
	Mtu         uint32
	Metric      uint32
	Baudrate    uint32
	Ipackets    uint32
	Ierrors     uint32
	Opackets    uint32
	Oerrors     uint32
	Collisions  uint32
	Ibytes      uint32
	Obytes      uint32
	Imcasts     uint32
	Omcasts     uint32
	Iqdrops     uint32
	Noproto     uint32
	Hwassist    uint32
	Pad_cgo_0   [4]byte
	Epoch       int64
	Lastchange  Timeval
}

type IfaMsghdr struct {
	Msglen    uint16
	Version   uint8
	Type      uint8
	Addrs     int32
	Flags     int32
	Index     uint16
	Pad_cgo_0 [2]byte
	Metric    int32
}

type IfmaMsghdr struct {
	Msglen    uint16
	Version   uint8
	Type      uint8
	Addrs     int32
	Flags     int32
	Index     uint16
	Pad_cgo_0 [2]byte
}

type IfAnnounceMsghdr struct {
	Msglen  uint16
	Version uint8
	Type    uint8
	Index   uint16
	Name    [16]int8
	What    uint16
}

type RtMsghdr struct {
	Msglen    uint16
	Version   uint8
	Type      uint8
	Index     uint16
	Pad_cgo_0 [2]byte
	Flags     int32
	Addrs     int32
	Pid       int32
	Seq       int32
	Errno     int32
	Fmask     int32
	Inits     uint32
	Rmx       RtMetrics
}

type RtMetrics struct {
	Locks    uint32
	Mtu      uint32
	Hopcount uint32
	Expire   uint32
	Recvpipe uint32
	Sendpipe uint32
	Ssthresh uint32
	Rtt      uint32
	Rttvar   uint32
	Pksent   uint32
	Weight   uint32
	Filler   [3]uint32
}

const (
	SizeofBpfVersion    = 0x4
	SizeofBpfStat       = 0x8
	SizeofBpfZbuf       = 0xc
	SizeofBpfProgram    = 0x8
	SizeofBpfInsn       = 0x8
	SizeofBpfHdr        = 0x20
	SizeofBpfZbufHeader = 0x20
)

type BpfVersion struct {
	Major uint16
	Minor uint16
}

type BpfStat struct {
	Recv uint32
	Drop uint32
}

type BpfZbuf struct {
	Bufa   *byte
	Bufb   *byte
	Buflen uint32
}

type BpfProgram struct {
	Len   uint32
	Insns *BpfInsn
}

type BpfInsn struct {
	Code uint16
	Jt   uint8
	Jf   uint8
	K    uint32
}

type BpfHdr struct {
	Tstamp    Timeval
	Caplen    uint32
	Datalen   uint32
	Hdrlen    uint16
	Pad_cgo_0 [6]byte
}

type BpfZbufHeader struct {
	Kernel_gen uint32
	Kernel_len uint32
	User_gen   uint32
	X_bzh_pad  [5]uint32
}

type Termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]uint8
	Ispeed uint32
	Ospeed uint32
}

type Winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

const (
	AT_FDCWD            = -0x64
	AT_REMOVEDIR        = 0x800
	AT_SYMLINK_FOLLOW   = 0x400
	AT_SYMLINK_NOFOLLOW = 0x200
)

type PollFd struct {
	Fd      int32
	Events  int16
	Revents int16
}

const (
	POLLERR      = 0x8
	POLLHUP      = 0x10
	POLLIN       = 0x1
	POLLINIGNEOF = 0x2000
	POLLNVAL     = 0x20
	POLLOUT      = 0x4
	POLLPRI      = 0x2
	POLLRDBAND   = 0x80
	POLLRDNORM   = 0x40
	POLLWRBAND   = 0x100
	POLLWRNORM   = 0x4
)

type CapRights struct {
	Rights [2]uint64
}
