package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit

#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>

static void renderEmojiToPNG(const char *emojiUTF8, const char *outPath) {
    @autoreleasepool {
        int size = 256;
        NSString *emoji = [NSString stringWithUTF8String:emojiUTF8];
        NSImage *image = [[NSImage alloc] initWithSize:NSMakeSize(size, size)];
        [image lockFocus];
        [[NSColor clearColor] set];
        NSRectFill(NSMakeRect(0, 0, size, size));
        CGFloat fontSize = size * 0.82;
        NSFont *font = [NSFont systemFontOfSize:fontSize];
        NSDictionary *attrs = @{NSFontAttributeName: font};
        NSSize strSize = [emoji sizeWithAttributes:attrs];
        NSPoint origin = NSMakePoint(
            (size - strSize.width) / 2.0,
            (size - strSize.height) / 2.0
        );
        [emoji drawAtPoint:origin withAttributes:attrs];
        [image unlockFocus];
        NSData *tiff = [image TIFFRepresentation];
        NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiff];
        NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG
                                       properties:@{}];
        [png writeToFile:[NSString stringWithUTF8String:outPath]
              atomically:YES];
    }
}
*/
import "C"

import (
	"os"
	"path/filepath"
	"unsafe"
)

func iconCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "emajor", "icons")
}

func iconCachePath(alias string) string {
	return filepath.Join(iconCacheDir(), alias+".png")
}

// ensureIcons generates any missing emoji PNGs and returns the path for each.
// Already-cached icons are a no-op. All generation happens in the current
// process via CGo/AppKit — no subprocess, no Python.
func ensureIcons(emojis []*Emoji) {
	dir := iconCacheDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	for _, e := range emojis {
		path := iconCachePath(e.Alias)
		if _, err := os.Stat(path); err == nil {
			continue // already cached
		}
		cEmoji := C.CString(e.Char)
		cPath := C.CString(path)
		C.renderEmojiToPNG(cEmoji, cPath)
		C.free(unsafe.Pointer(cEmoji))
		C.free(unsafe.Pointer(cPath))
	}
}
