package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	dockerclient "github.com/docker/docker/client"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imgtype "github.com/docker/docker/api/types/image"
	nettype "github.com/docker/docker/api/types/network"
	voltype "github.com/docker/docker/api/types/volume"
)

type Client struct {
	cli *dockerclient.Client
}

func NewClient() (*Client, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("cannot connect to Docker daemon: %w", err)
	}

	return &Client{cli: cli}, nil
}

func (c *Client) Close() {
	c.cli.Close()
}

// Containers

func (c *Client) ListContainers(ctx context.Context, all bool) ([]Container, error) {
	list, err := c.cli.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, err
	}

	out := make([]Container, 0, len(list))
	for _, ct := range list {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		out = append(out, Container{
			ID:      shortID(ct.ID),
			FullID:  ct.ID,
			Name:    name,
			Image:   ct.Image,
			Status:  ct.Status,
			State:   ct.State,
			Ports:   formatPorts(ct.Ports),
			Created: time.Unix(ct.Created, 0),
		})
	}
	return out, nil
}

func (c *Client) StopContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStop(ctx, id, container.StopOptions{})
}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (c *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	return c.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

func (c *Client) FetchLogs(ctx context.Context, id string, tail string) (string, error) {
	if tail == "" {
		tail = "500"
	}
	rc, err := c.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		return "", err
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	// Strip docker log multiplexing headers (8-byte prefix per line)
	return stripLogHeaders(b), nil
}

func (c *Client) InspectContainer(ctx context.Context, id string) (string, error) {
	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ID:           %s\n", info.ID[:12]))
	sb.WriteString(fmt.Sprintf("Full ID:      %s\n", info.ID))
	sb.WriteString(fmt.Sprintf("Name:         %s\n", strings.TrimPrefix(info.Name, "/")))
	sb.WriteString(fmt.Sprintf("Image:        %s\n", info.Config.Image))
	sb.WriteString(fmt.Sprintf("Status:       %s\n", info.State.Status))
	sb.WriteString(fmt.Sprintf("Created:      %s\n", info.Created))
	sb.WriteString(fmt.Sprintf("Restart Policy: %s\n", info.HostConfig.RestartPolicy.Name))

	if info.NetworkSettings != nil {
		for netName, net := range info.NetworkSettings.Networks {
			sb.WriteString(fmt.Sprintf("Network:      %s (%s)\n", netName, net.IPAddress))
		}
	}

	if len(info.Config.Cmd) > 0 {
		sb.WriteString(fmt.Sprintf("Cmd:          %s\n", strings.Join(info.Config.Cmd, " ")))
	}
	if len(info.Config.Entrypoint) > 0 {
		sb.WriteString(fmt.Sprintf("Entrypoint:   %s\n", strings.Join(info.Config.Entrypoint, " ")))
	}

	if len(info.Mounts) > 0 {
		sb.WriteString("\nMounts:\n")
		for _, m := range info.Mounts {
			sb.WriteString(fmt.Sprintf("  %s -> %s (%s)\n", m.Source, m.Destination, m.Mode))
		}
	}

	if len(info.Config.Env) > 0 {
		sb.WriteString("\nEnvironment:\n")
		for _, e := range info.Config.Env {
			sb.WriteString(fmt.Sprintf("  %s\n", e))
		}
	}

	if len(info.Config.ExposedPorts) > 0 {
		sb.WriteString("\nExposed Ports:\n")
		for p := range info.Config.ExposedPorts {
			sb.WriteString(fmt.Sprintf("  %s\n", p))
		}
	}

	if len(info.Config.Labels) > 0 {
		sb.WriteString("\nLabels:\n")
		for k, v := range info.Config.Labels {
			sb.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
	}

	return sb.String(), nil
}

// Volumes

func (c *Client) ListVolumes(ctx context.Context) ([]Volume, error) {
	resp, err := c.cli.VolumeList(ctx, voltype.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Fetch disk usage to get sizes
	sizes := map[string]voltype.UsageData{}
	du, err := c.cli.DiskUsage(ctx, types.DiskUsageOptions{Types: []types.DiskUsageObject{types.VolumeObject}})
	if err == nil {
		for _, v := range du.Volumes {
			if v.UsageData != nil {
				sizes[v.Name] = *v.UsageData
			}
		}
	}

	out := make([]Volume, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		vol := Volume{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Scope:      v.Scope,
			CreatedAt:  v.CreatedAt,
			Size:       -1,
			RefCount:   -1,
			Labels:     v.Labels,
		}
		if ud, ok := sizes[v.Name]; ok {
			vol.Size = ud.Size
			vol.RefCount = ud.RefCount
		}
		out = append(out, vol)
	}
	return out, nil
}

func (c *Client) GetContainersForVolume(ctx context.Context, volumeName string) ([]Container, error) {
	list, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	var out []Container
	for _, ct := range list {
		for _, m := range ct.Mounts {
			if m.Name == volumeName {
				name := ""
				if len(ct.Names) > 0 {
					name = strings.TrimPrefix(ct.Names[0], "/")
				}
				out = append(out, Container{
					ID:     shortID(ct.ID),
					FullID: ct.ID,
					Name:   name,
					Image:  ct.Image,
					Status: ct.Status,
					State:  ct.State,
				})
				break
			}
		}
	}
	return out, nil
}

func (c *Client) BackupVolume(ctx context.Context, volumeName, destPath string) error {
	absDestPath, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("resolving destination path: %w", err)
	}
	destDir := filepath.Dir(absDestPath)
	destFile := filepath.Base(absDestPath)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	// Pull alpine (best-effort; if offline, use whatever is cached locally).
	if rc, pullErr := c.cli.ImagePull(ctx, "alpine", imgtype.PullOptions{}); pullErr == nil {
		io.Copy(io.Discard, rc)
		rc.Close()
	}

	resp, err := c.cli.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine",
			Cmd:   []string{"tar", "czf", "/backup_dest/" + destFile, "-C", "/backup_source", "."},
		},
		&container.HostConfig{
			Binds: []string{
				volumeName + ":/backup_source:ro",
				destDir + ":/backup_dest",
			},
		},
		nil, nil, "")
	if err != nil {
		return fmt.Errorf("creating backup container: %w", err)
	}
	defer c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting backup container: %w", err)
	}

	statusCh, errCh := c.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("waiting for backup container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("backup container exited with code %d", status.StatusCode)
		}
	}
	return nil
}

func (c *Client) RestoreVolume(ctx context.Context, volumeName, sourcePath string, replace bool) error {
	absSrcPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("resolving source path: %w", err)
	}
	srcDir := filepath.Dir(absSrcPath)
	srcFile := filepath.Base(absSrcPath)

	// Pull alpine (best-effort).
	if rc, pullErr := c.cli.ImagePull(ctx, "alpine", imgtype.PullOptions{}); pullErr == nil {
		io.Copy(io.Discard, rc)
		rc.Close()
	}

	var cmd []string
	if replace {
		cmd = []string{"sh", "-c",
			"find /restore_dest -mindepth 1 -delete && tar xzf /restore_source/" + srcFile + " -C /restore_dest",
		}
	} else {
		cmd = []string{"tar", "xzf", "/restore_source/" + srcFile, "-C", "/restore_dest"}
	}

	resp, err := c.cli.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine",
			Cmd:   cmd,
		},
		&container.HostConfig{
			Binds: []string{
				volumeName + ":/restore_dest",
				srcDir + ":/restore_source:ro",
			},
		},
		nil, nil, "")
	if err != nil {
		return fmt.Errorf("creating restore container: %w", err)
	}
	defer c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting restore container: %w", err)
	}

	statusCh, errCh := c.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("waiting for restore container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("restore container exited with code %d", status.StatusCode)
		}
	}
	return nil
}

func (c *Client) RemoveVolume(ctx context.Context, name string, force bool) error {
	return c.cli.VolumeRemove(ctx, name, force)
}

func (c *Client) InspectVolume(ctx context.Context, name string) (string, error) {
	v, err := c.cli.VolumeInspect(ctx, name)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name:       %s\n", v.Name))
	sb.WriteString(fmt.Sprintf("Driver:     %s\n", v.Driver))
	sb.WriteString(fmt.Sprintf("Scope:      %s\n", v.Scope))
	sb.WriteString(fmt.Sprintf("Mountpoint: %s\n", v.Mountpoint))
	sb.WriteString(fmt.Sprintf("Created:    %s\n", v.CreatedAt))

	if v.UsageData != nil {
		sb.WriteString(fmt.Sprintf("Size:       %s\n", formatBytes(v.UsageData.Size)))
		sb.WriteString(fmt.Sprintf("Ref Count:  %d\n", v.UsageData.RefCount))
	}

	if len(v.Options) > 0 {
		sb.WriteString("\nOptions:\n")
		for k, val := range v.Options {
			sb.WriteString(fmt.Sprintf("  %s=%s\n", k, val))
		}
	}

	if len(v.Labels) > 0 {
		sb.WriteString("\nLabels:\n")
		for k, val := range v.Labels {
			sb.WriteString(fmt.Sprintf("  %s=%s\n", k, val))
		}
	}

	return sb.String(), nil
}

// Networks

func (c *Client) ListNetworks(ctx context.Context) ([]Network, error) {
	list, err := c.cli.NetworkList(ctx, nettype.ListOptions{})
	if err != nil {
		return nil, err
	}

	out := make([]Network, 0, len(list))
	for _, n := range list {
		net := Network{
			ID:       shortID(n.ID),
			Name:     n.Name,
			Driver:   n.Driver,
			Scope:    n.Scope,
			Internal: n.Internal,
			IPv6:     n.EnableIPv6,
			Labels:   n.Labels,
		}
		if n.IPAM.Config != nil && len(n.IPAM.Config) > 0 {
			net.Subnet = n.IPAM.Config[0].Subnet
			net.Gateway = n.IPAM.Config[0].Gateway
		}
		for name := range n.Containers {
			net.Containers = append(net.Containers, name)
		}
		out = append(out, net)
	}
	return out, nil
}

func (c *Client) RemoveNetwork(ctx context.Context, id string) error {
	return c.cli.NetworkRemove(ctx, id)
}

func (c *Client) InspectNetwork(ctx context.Context, id string) (string, error) {
	n, err := c.cli.NetworkInspect(ctx, id, nettype.InspectOptions{})
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ID:       %s\n", shortID(n.ID)))
	sb.WriteString(fmt.Sprintf("Full ID:  %s\n", n.ID))
	sb.WriteString(fmt.Sprintf("Name:     %s\n", n.Name))
	sb.WriteString(fmt.Sprintf("Driver:   %s\n", n.Driver))
	sb.WriteString(fmt.Sprintf("Scope:    %s\n", n.Scope))
	sb.WriteString(fmt.Sprintf("Internal: %v\n", n.Internal))
	sb.WriteString(fmt.Sprintf("IPv6:     %v\n", n.EnableIPv6))
	sb.WriteString(fmt.Sprintf("Created:  %s\n", n.Created.Format(time.RFC3339)))

	if n.IPAM.Config != nil && len(n.IPAM.Config) > 0 {
		sb.WriteString("\nIPAM:\n")
		for _, cfg := range n.IPAM.Config {
			if cfg.Subnet != "" {
				sb.WriteString(fmt.Sprintf("  Subnet:  %s\n", cfg.Subnet))
			}
			if cfg.Gateway != "" {
				sb.WriteString(fmt.Sprintf("  Gateway: %s\n", cfg.Gateway))
			}
		}
	}

	if len(n.Containers) > 0 {
		sb.WriteString("\nContainers:\n")
		for _, ct := range n.Containers {
			sb.WriteString(fmt.Sprintf("  %s (%s)\n", ct.Name, ct.IPv4Address))
		}
	}

	if len(n.Options) > 0 {
		sb.WriteString("\nOptions:\n")
		for k, v := range n.Options {
			sb.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
	}

	if len(n.Labels) > 0 {
		sb.WriteString("\nLabels:\n")
		for k, v := range n.Labels {
			sb.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
	}

	return sb.String(), nil
}

// Images

func (c *Client) ListImages(ctx context.Context) ([]Image, error) {
	list, err := c.cli.ImageList(ctx, imgtype.ListOptions{All: false})
	if err != nil {
		return nil, err
	}

	out := make([]Image, 0, len(list))
	for _, img := range list {
		id := img.ID
		if strings.HasPrefix(id, "sha256:") {
			id = id[7:]
		}
		out = append(out, Image{
			ID:      id[:min(12, len(id))],
			FullID:  img.ID,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: time.Unix(img.Created, 0),
		})
	}
	return out, nil
}

func (c *Client) RemoveImage(ctx context.Context, id string, force bool) error {
	_, err := c.cli.ImageRemove(ctx, id, imgtype.RemoveOptions{Force: force})
	return err
}

func (c *Client) InspectImage(ctx context.Context, id string) (string, error) {
	img, err := c.cli.ImageInspect(ctx, id)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	imgID := img.ID
	if strings.HasPrefix(imgID, "sha256:") {
		imgID = imgID[7:]
	}
	sb.WriteString(fmt.Sprintf("ID:           %s\n", imgID[:12]))
	sb.WriteString(fmt.Sprintf("Full ID:      %s\n", img.ID))

	if len(img.RepoTags) > 0 {
		sb.WriteString("Tags:\n")
		for _, t := range img.RepoTags {
			sb.WriteString(fmt.Sprintf("  %s\n", t))
		}
	}
	if len(img.RepoDigests) > 0 {
		sb.WriteString("Digests:\n")
		for _, d := range img.RepoDigests {
			sb.WriteString(fmt.Sprintf("  %s\n", d))
		}
	}

	sb.WriteString(fmt.Sprintf("Size:         %s\n", formatBytes(img.Size)))
	sb.WriteString(fmt.Sprintf("Created:      %s\n", img.Created))
	sb.WriteString(fmt.Sprintf("Architecture: %s\n", img.Architecture))
	sb.WriteString(fmt.Sprintf("OS:           %s\n", img.Os))

	if img.Config != nil {
		if len(img.Config.Cmd) > 0 {
			sb.WriteString(fmt.Sprintf("Cmd:          %s\n", strings.Join(img.Config.Cmd, " ")))
		}
		if len(img.Config.Entrypoint) > 0 {
			sb.WriteString(fmt.Sprintf("Entrypoint:   %s\n", strings.Join(img.Config.Entrypoint, " ")))
		}
		if len(img.Config.ExposedPorts) > 0 {
			sb.WriteString("Exposed Ports:\n")
			for p := range img.Config.ExposedPorts {
				sb.WriteString(fmt.Sprintf("  %s\n", p))
			}
		}
		if len(img.Config.Labels) > 0 {
			sb.WriteString("Labels:\n")
			for k, v := range img.Config.Labels {
				sb.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
			}
		}
	}

	return sb.String(), nil
}

// Helpers

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func formatPorts(ports []types.Port) string {
	var parts []string
	seen := make(map[string]bool)
	for _, p := range ports {
		var s string
		if p.IP != "" && p.PublicPort != 0 {
			s = fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type)
		} else if p.PublicPort != 0 {
			s = fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type)
		} else {
			s = fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
		}
		if !seen[s] {
			seen[s] = true
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", ")
}

func formatBytes(b int64) string {
	if b < 0 {
		return "unknown"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// stripLogHeaders removes the 8-byte multiplexing header Docker prepends to each log line.
func stripLogHeaders(b []byte) string {
	var sb strings.Builder
	for len(b) >= 8 {
		// Header: [stream_type(1), 0, 0, 0, size(4 big-endian)]
		size := int(b[4])<<24 | int(b[5])<<16 | int(b[6])<<8 | int(b[7])
		b = b[8:]
		if size > len(b) {
			size = len(b)
		}
		sb.Write(b[:size])
		b = b[size:]
	}
	return sb.String()
}

// ListOptions alias for filter helpers
var _ = filters.Args{}
