# Build Instructions

## macOS Codesigning

VMTerminal uses macOS Virtualization.framework which requires entitlements.

After building, codesign the binary:

```bash
codesign --entitlements build/vz.entitlements -s - ./vmterminal
```

Without codesigning, the binary will crash with "not entitled to use Virtualization".

## Linux

No special build steps required. Ensure /dev/kvm is accessible (user in kvm group).
