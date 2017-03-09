package machine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"koding/klient/machine"
	"koding/klient/machine/machinegroup"
	"koding/klient/machine/mount"
	"koding/klient/machine/mount/sync"
	"koding/klient/machine/transport/rsync"
	"koding/klientctl/klient"

	"github.com/dustin/go-humanize"
	"github.com/koding/logging"
	"github.com/mitchellh/ioprogress"
)

// MountOptions stores options for `machine mount` call.
type MountOptions struct {
	Identifier string // Machine identifier.
	Path       string // Machine local path - absolute and cleaned.
	RemotePath string // Remote machine path - raw format.
	Log        logging.Logger
}

// Mount synchronizes directories between remote and local machines.
func Mount(options *MountOptions) (err error) {
	if options == nil {
		return errors.New("invalid nil options")
	}

	// Create and check mount point directory.
	clean, err := mountPointDirectory(options.Path)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			clean()
		}
	}()

	// TODO(ppknap): this is copied from klientctl old list and will be reworked.
	k, err := klient.CreateKlientWithDefaultOpts()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating klient:", err)
		return err
	}

	if err := k.Dial(); err != nil {
		return err
	}

	// Translate identifier to machine ID.
	idReq := machinegroup.IDRequest{
		Identifier: options.Identifier,
	}
	idRaw, err := k.Tell("machine.id", idReq)
	if err != nil {
		return err
	}
	idRes := machinegroup.IDResponse{}
	if err := idRaw.Unmarshal(&idRes); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Mounting to %s directory.\nChecking remote path...\n", options.Path)

	m := mount.Mount{
		Path:       options.Path,
		RemotePath: options.RemotePath,
	}
	// First head the remote machine directory in order to get basic mount info.
	headMountReq := machinegroup.HeadMountRequest{
		MountRequest: machinegroup.MountRequest{
			ID:    idRes.ID,
			Mount: m,
		},
	}
	headMountRaw, err := k.Tell("machine.mount.head", headMountReq)
	if err != nil {
		return err
	}
	headMountRes := machinegroup.HeadMountResponse{}
	if err := headMountRaw.Unmarshal(&headMountRes); err != nil {
		return err
	}

	// Remote directory is already mounted to this machine.
	//
	// TODO: ask user if she wants to create another mount or stop the process.
	if headMountRes.ExistMountID != "" {
		fmt.Fprintf(os.Stdout, "Remote directory %s is already mounted by: %s\n",
			headMountRes.AbsRemotePath, headMountRes.ExistMountID)

		clean()
		return nil
	}

	fmt.Fprintf(os.Stdout, "Mounted remote directory %s has %d file(s) of total size %s\n",
		headMountRes.AbsRemotePath, headMountRes.AllCount, humanize.IBytes(uint64(headMountRes.AllDiskSize)))

	// TODO: ask user if she wants to continue.

	m.RemotePath = headMountRes.AbsRemotePath
	fmt.Fprintf(os.Stdout, "Initializing mount %s...\n", m)

	// Create mount.
	addMountReq := machinegroup.AddMountRequest{
		MountRequest: machinegroup.MountRequest{
			ID:    idRes.ID,
			Mount: m,
		},
	}
	addMountRaw, err := k.Tell("machine.mount.add", addMountReq)
	if err != nil {
		return err
	}
	addMountRes := machinegroup.AddMountResponse{}
	if err := addMountRaw.Unmarshal(&addMountRes); err != nil {
		return err
	}

	// Prefetch files.
	if _, _, privPath, err := sshGetKeyPath(); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot prefetch mount files: %s\n", err)
	} else if addMountRes.SourcePath != "" && addMountRes.DestinationPath != "" {
		cmd := &rsync.Command{
			Download:        true,
			SourcePath:      addMountRes.SourcePath,
			DestinationPath: addMountRes.DestinationPath,
			Username:        addMountRes.Username,
			Host:            addMountRes.Host,
			SSHPort:         addMountRes.SSHPort,
			PrivateKeyPath:  privPath,
			Progress:        drawProgress(os.Stdout, addMountRes.Count, addMountRes.DiskSize),
		}

		if err := cmd.Run(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "File prefetching interrupted: %s\n", err)
		}
	}

	fmt.Fprintf(os.Stdout, "Created mount with ID: %s\n", addMountRes.MountID)
	return nil
}

// ListMountOptions stores options for `machine mount list` call.
type ListMountOptions struct {
	ID      string // Machine ID - optional.
	MountID string // Mount ID - optional.
	Log     logging.Logger
}

// ListMount removes existing mount.
func ListMount(options *ListMountOptions) (map[string][]sync.Info, error) {
	if options == nil {
		return nil, errors.New("invalid nil options")
	}

	// TODO(ppknap): this is copied from klientctl old list and will be reworked.
	k, err := klient.CreateKlientWithDefaultOpts()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating klient:", err)
		return nil, err
	}

	if err := k.Dial(); err != nil {
		return nil, err
	}

	// List mounts.
	listMountReq := machinegroup.ListMountRequest{
		ID:      machine.ID(options.ID),
		MountID: mount.ID(options.MountID),
	}
	listMountRaw, err := k.Tell("machine.mount.list", listMountReq)
	if err != nil {
		return nil, err
	}
	listMountRes := machinegroup.ListMountResponse{}
	if err := listMountRaw.Unmarshal(&listMountRes); err != nil {
		return nil, err
	}

	return listMountRes.Mounts, nil
}

// UmountOptions stores options for `machine umount` call.
type UmountOptions struct {
	Identifier string // Mount identifier.
	Log        logging.Logger
}

// Umount removes existing mount.
func Umount(options *UmountOptions) (err error) {
	if options == nil {
		return errors.New("invalid nil options")
	}

	// TODO(ppknap): this is copied from klientctl old list and will be reworked.
	k, err := klient.CreateKlientWithDefaultOpts()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating klient:", err)
		return err
	}

	if err := k.Dial(); err != nil {
		return err
	}

	// TODO: ask user to confirm unmounting.
	fmt.Fprintf(os.Stdout, "Unmounting %s...\n", options.Identifier)

	// Remove mount.
	umountReq := machinegroup.UmountRequest{
		Identifier: options.Identifier,
	}
	umountRaw, err := k.Tell("machine.umount", umountReq)
	if err != nil {
		return err
	}
	umountRes := machinegroup.UmountResponse{}
	if err := umountRaw.Unmarshal(&umountRes); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Successfully unmounted %s (ID: %s)\n",
		umountRes.Mount, umountRes.MountID)

	return nil
}

// mountPointDirectory checks and prepares local directory for mounting.
// Returned clean function can be used to remove resources in case of other
// mounting errors.
//
// NOTE: This logic will be moved to klient.
func mountPointDirectory(path string) (clean func(), err error) {
	switch info, err := os.Stat(path); {
	case os.IsNotExist(err):
		// Create a new directory.
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("cannot create destination directory: %s", err)
		}
		return func() {
			os.RemoveAll(path)
		}, nil
	case err != nil:
		return nil, fmt.Errorf("cannot stat destination directory: %s", err)
	case !info.IsDir():
		return nil, fmt.Errorf("file %q is not a directory", path)

	}

	// Provided directory already exists. Check if it's empty.
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open destination directory: %s", err)
	}
	defer f.Close()

	switch _, err = f.Readdirnames(1); err {
	case nil:
		return nil, errors.New("destination directory is not empty")
	case io.EOF:
	default:
		return nil, fmt.Errorf("destination directory error: %s", err)
	}
	clean = func() {
		removeContent(path)
	}

	return clean, nil

}

// removeContent removes all files inside provided path but not the path itself.
func removeContent(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		names, err := f.Readdirnames(100)
		if err != nil {
			return err
		}

		for _, name := range names {
			os.RemoveAll(filepath.Join(path, name)) // Ignore errors.
		}
	}
}

func drawProgress(w io.Writer, nAll, sizeAll int64) func(n, size, speed int64, err error) {
	const noop = 0
	return func(n, size, speed int64, err error) {
		drawFunc := ioprogress.DrawTerminalf(w, func(_, _ int64) string {
			return fmt.Sprintf("Prefetching files: %d%% (%d/%d), %s/%s | %s/s",
				int(float64(n)/float64(nAll)*100.0+0.5), // percentage status.
				n,    // number of downloaded files.
				nAll, // number of all files being downloaded.
				humanize.IBytes(uint64(size)),    // size of downloaded files.
				humanize.IBytes(uint64(sizeAll)), // total size.
				humanize.IBytes(uint64(speed)),   // current downloading speed.
			)
		})

		if err == io.EOF {
			drawFunc(-1, -1) // Finish drawing.
			return
		}

		drawFunc(noop, noop) // We are not using default values.
	}
}
