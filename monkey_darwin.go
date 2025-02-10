//go:build darwin

package monkey

import (
	"syscall"
)

const (
	_KERN_SUCCESS         = 0
	_SYS_MACH_TRAP_UNUSED = 0x1000000 // Base for Mach trap numbers
	_MACH_VM_PROTECT      = 14        // Mach trap number for vm_protect
)

// MachVmProtect is the Mach trap number for vm_protect
const _MachVmProtect = 0x200000

func machCall(h uint32, args ...uint64) (uint32, error) {
	ret, _, err := syscall.RawSyscall6(
		uintptr(_SYS_MACH_TRAP_UNUSED+_MACH_VM_PROTECT),
		uintptr(h),
		uintptr(args[0]),
		uintptr(args[1]),
		uintptr(args[2]),
		uintptr(args[3]),
		0, // Last argument can be 0
	)
	if err != 0 {
		return uint32(ret), err
	}
	return uint32(ret), nil
}

func makeWritableDarwin(addr uintptr, length int) error {
	pageSize := syscall.Getpagesize()
	pageStart := addr & ^(uintptr(pageSize - 1))

	ret, err := machCall(0xffffffff, // mach_task_self
		uint64(pageStart),
		uint64(pageSize),
		0, // inherit current protection
		7, // RWX: 1 | 2 | 4
		0) // reserved

	if err != nil {
		return err
	}
	if ret != _KERN_SUCCESS {
		return syscall.EPERM
	}
	return nil
}
