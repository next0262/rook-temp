#!/usr/bin/env -S bash -e

OSD_COUNT=3
for i in $(seq 1 $OSD_COUNT); do
    DISK="/dev/nvme${i}n1"
    echo "###Start to wipe for ${DISK}###"
    # Zap the disk to a fresh, usable state (zap-all is important, b/c MBR has to be clean)
    sgdisk --zap-all $DISK

    # Wipe a large portion of the beginning of the disk to remove more LVM metadata that may be present
    dd if=/dev/zero of="$DISK" bs=1M count=100 oflag=direct,dsync

    # SSDs may be better cleaned with blkdiscard instead of dd
    blkdiscard $DISK

    # Inform the OS of partition table changes
    partprobe $DISK
done
