//go:build darwin && cgo

package cli

/*
#cgo CFLAGS: -x objective-c -Wno-unused-variable
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void cu_setDockIcon(const void *data, int len) {
    @autoreleasepool {
        NSData *d = [NSData dataWithBytes:data length:len];
        NSImage *img = [[NSImage alloc] initWithData:d];
        if (img != nil) {
            dispatch_async(dispatch_get_main_queue(), ^{
                [NSApp setApplicationIconImage:img];
            });
        }
    }
}

static void cu_showInDock(int show) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (show) {
            [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        } else {
            [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        }
    });
}
*/
import "C"
import "unsafe"

// dockSupported reports whether updating a Dock icon is possible in
// this build (true on macOS with CGO).
func dockSupported() bool { return true }

// dockSetIcon replaces the app's Dock icon with the given PNG bytes.
// Must be called after NSApp is running (i.e. after systray.Run kicks
// off). The call is dispatched to the main thread internally.
func dockSetIcon(pngBytes []byte) {
	if len(pngBytes) == 0 {
		return
	}
	C.cu_setDockIcon(unsafe.Pointer(&pngBytes[0]), C.int(len(pngBytes)))
}

// dockShow toggles whether the app appears in the Dock at all.
// macOS apps are NSApplicationActivationPolicyAccessory by default when
// embedded via systray; switch to Regular to show a Dock icon (and
// therefore the progress-bar image we paint onto it).
func dockShow(show bool) {
	v := C.int(0)
	if show {
		v = 1
	}
	C.cu_showInDock(v)
}
