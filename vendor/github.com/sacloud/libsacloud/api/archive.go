package api

import (
	"fmt"
	"github.com/sacloud/libsacloud/sacloud"
	"github.com/sacloud/libsacloud/sacloud/ostype"
	"strings"
	"time"
)

// ArchiveAPI アーカイブAPI
type ArchiveAPI struct {
	*baseAPI
	findFuncMapPerOSType map[ostype.ArchiveOSTypes]func() (*sacloud.Archive, error)
}

var (
	archiveLatestStableCentOSTags                      = []string{"current-stable", "distro-centos"}
	archiveLatestStableUbuntuTags                      = []string{"current-stable", "distro-ubuntu"}
	archiveLatestStableDebianTags                      = []string{"current-stable", "distro-debian"}
	archiveLatestStableVyOSTags                        = []string{"current-stable", "distro-vyos"}
	archiveLatestStableCoreOSTags                      = []string{"current-stable", "distro-coreos"}
	archiveLatestStableRancherOSTags                   = []string{"current-stable", "distro-rancheros"}
	archiveLatestStableKusanagiTags                    = []string{"current-stable", "pkg-kusanagi"}
	archiveLatestStableSiteGuardTags                   = []string{"current-stable", "pkg-siteguard"}
	archiveLatestStablePleskTags                       = []string{"current-stable", "pkg-plesk"}
	archiveLatestStableFreeBSDTags                     = []string{"current-stable", "distro-freebsd"}
	archiveLatestStableWindows2012Tags                 = []string{"os-windows", "distro-ver-2012.2"}
	archiveLatestStableWindows2012RDSTags              = []string{"os-windows", "distro-ver-2012.2", "windows-rds"}
	archiveLatestStableWindows2012RDSOfficeTags        = []string{"os-windows", "distro-ver-2012.2", "windows-rds", "with-office"}
	archiveLatestStableWindows2016Tags                 = []string{"os-windows", "distro-ver-2016"}
	archiveLatestStableWindows2016RDSTags              = []string{"os-windows", "distro-ver-2016", "windows-rds"}
	archiveLatestStableWindows2016RDSOfficeTags        = []string{"os-windows", "distro-ver-2016", "windows-rds", "with-office"}
	archiveLatestStableWindows2016SQLServerWeb         = []string{"os-windows", "distro-ver-2016", "windows-sqlserver", "sqlserver-2016", "edition-web"}
	archiveLatestStableWindows2016SQLServerStandard    = []string{"os-windows", "distro-ver-2016", "windows-sqlserver", "sqlserver-2016", "edition-standard"}
	archiveLatestStableWindows2016SQLServerStandardAll = []string{"os-windows", "distro-ver-2016", "windows-sqlserver", "sqlserver-2016", "edition-standard", "windows-rds", "with-office"}
)

// NewArchiveAPI アーカイブAPI作成
func NewArchiveAPI(client *Client) *ArchiveAPI {
	api := &ArchiveAPI{
		baseAPI: &baseAPI{
			client: client,
			FuncGetResourceURL: func() string {
				return "archive"
			},
		},
	}

	api.findFuncMapPerOSType = map[ostype.ArchiveOSTypes]func() (*sacloud.Archive, error){
		ostype.CentOS:                          api.FindLatestStableCentOS,
		ostype.Ubuntu:                          api.FindLatestStableUbuntu,
		ostype.Debian:                          api.FindLatestStableDebian,
		ostype.VyOS:                            api.FindLatestStableVyOS,
		ostype.CoreOS:                          api.FindLatestStableCoreOS,
		ostype.RancherOS:                       api.FindLatestStableRancherOS,
		ostype.Kusanagi:                        api.FindLatestStableKusanagi,
		ostype.SiteGuard:                       api.FindLatestStableSiteGuard,
		ostype.Plesk:                           api.FindLatestStablePlesk,
		ostype.FreeBSD:                         api.FindLatestStableFreeBSD,
		ostype.Windows2012:                     api.FindLatestStableWindows2012,
		ostype.Windows2012RDS:                  api.FindLatestStableWindows2012RDS,
		ostype.Windows2012RDSOffice:            api.FindLatestStableWindows2012RDSOffice,
		ostype.Windows2016:                     api.FindLatestStableWindows2016,
		ostype.Windows2016RDS:                  api.FindLatestStableWindows2016RDS,
		ostype.Windows2016RDSOffice:            api.FindLatestStableWindows2016RDSOffice,
		ostype.Windows2016SQLServerWeb:         api.FindLatestStableWindows2016SQLServerWeb,
		ostype.Windows2016SQLServerStandard:    api.FindLatestStableWindows2016SQLServerStandard,
		ostype.Windows2016SQLServerStandardAll: api.FindLatestStableWindows2016SQLServerStandardAll,
	}

	return api
}

// OpenFTP FTP接続開始
func (api *ArchiveAPI) OpenFTP(id int64) (*sacloud.FTPServer, error) {
	var (
		method = "PUT"
		uri    = fmt.Sprintf("%s/%d/ftp", api.getResourceURL(), id)
		//body   = map[string]bool{"ChangePassword": reset}
		res = &sacloud.Response{}
	)

	result, err := api.action(method, uri, nil, res)
	if !result || err != nil {
		return nil, err
	}

	return res.FTPServer, nil
}

// CloseFTP FTP接続終了
func (api *ArchiveAPI) CloseFTP(id int64) (bool, error) {
	var (
		method = "DELETE"
		uri    = fmt.Sprintf("%s/%d/ftp", api.getResourceURL(), id)
	)
	return api.modify(method, uri, nil)

}

// SleepWhileCopying コピー終了まで待機
func (api *ArchiveAPI) SleepWhileCopying(id int64, timeout time.Duration) error {

	current := 0 * time.Second
	interval := 5 * time.Second
	for {
		archive, err := api.Read(id)
		if err != nil {
			return err
		}

		if archive.IsAvailable() {
			return nil
		}
		time.Sleep(interval)
		current += interval

		if timeout > 0 && current > timeout {
			return fmt.Errorf("Timeout: SleepWhileCopying[disk:%d]", id)
		}
	}
}

// AsyncSleepWhileCopying コピー終了まで待機(非同期)
func (api *ArchiveAPI) AsyncSleepWhileCopying(id int64, timeout time.Duration) (chan (*sacloud.Archive), chan (*sacloud.Archive), chan (error)) {
	complete := make(chan *sacloud.Archive)
	progress := make(chan *sacloud.Archive)
	err := make(chan error)

	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				archive, e := api.Read(id)
				if e != nil {
					err <- e
					return
				}

				progress <- archive

				if archive.IsAvailable() {
					complete <- archive
					return
				}
				if archive.IsFailed() {
					err <- fmt.Errorf("Failed: Create archive is failed: %#v", archive)
					return
				}

			case <-time.After(timeout):
				err <- fmt.Errorf("Timeout: AsyncSleepWhileCopying[ID:%d]", id)
				return
			}
		}
	}()
	return complete, progress, err
}

// CanEditDisk ディスクの修正が可能か判定
func (api *ArchiveAPI) CanEditDisk(id int64) (bool, error) {

	archive, err := api.Read(id)
	if err != nil {
		return false, err
	}

	if archive == nil {
		return false, nil
	}

	// BundleInfoがあれば編集不可
	if archive.BundleInfo != nil && archive.BundleInfo.HostClass == bundleInfoWindowsHostClass {
		// Windows
		return false, nil
	}

	for _, t := range allowDiskEditTags {
		if archive.HasTag(t) {
			// 対応OSインストール済みディスク
			return true, nil
		}
	}

	// ここまできても判定できないならソースに投げる
	if archive.SourceDisk != nil && archive.SourceDisk.Availability != "discontinued" {
		return api.client.Disk.CanEditDisk(archive.SourceDisk.ID)
	}
	if archive.SourceArchive != nil && archive.SourceArchive.Availability != "discontinued" {
		return api.client.Archive.CanEditDisk(archive.SourceArchive.ID)
	}
	return false, nil

}

// FindLatestStableCentOS 安定版最新のCentOSパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableCentOS() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableCentOSTags)
}

// FindLatestStableDebian 安定版最新のDebianパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableDebian() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableDebianTags)
}

// FindLatestStableUbuntu 安定版最新のUbuntuパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableUbuntu() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableUbuntuTags)
}

// FindLatestStableVyOS 安定版最新のVyOSパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableVyOS() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableVyOSTags)
}

// FindLatestStableCoreOS 安定版最新のCoreOSパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableCoreOS() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableCoreOSTags)
}

// FindLatestStableRancherOS 安定版最新のRancherOSパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableRancherOS() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableRancherOSTags)
}

// FindLatestStableKusanagi 安定版最新のKusanagiパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableKusanagi() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableKusanagiTags)
}

// FindLatestStableSiteGuard 安定版最新のSiteGuardパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableSiteGuard() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableSiteGuardTags)
}

// FindLatestStablePlesk 安定版最新のPleskパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStablePlesk() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStablePleskTags)
}

// FindLatestStableFreeBSD 安定版最新のFreeBSDパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableFreeBSD() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableFreeBSDTags)
}

// FindLatestStableWindows2012 安定版最新のWindows2012パブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2012() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2012Tags, map[string]interface{}{
		"Name": "Windows Server 2012 R2 Datacenter Edition",
	})
}

// FindLatestStableWindows2012RDS 安定版最新のWindows2012RDSパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2012RDS() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2012RDSTags, map[string]interface{}{
		"Name": "Windows Server 2012 R2 for RDS",
	})
}

// FindLatestStableWindows2012RDSOffice 安定版最新のWindows2012RDS(Office)パブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2012RDSOffice() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2012RDSOfficeTags, map[string]interface{}{
		"Name": "Windows Server 2012 R2 for RDS(MS Office付)",
	})
}

// FindLatestStableWindows2016 安定版最新のWindows2016パブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2016() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2016Tags, map[string]interface{}{
		"Name": "Windows Server 2016 Datacenter Edition",
	})
}

// FindLatestStableWindows2016RDS 安定版最新のWindows2016RDSパブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2016RDS() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2016RDSTags, map[string]interface{}{
		"Name": "Windows Server 2016 for RDS",
	})
}

// FindLatestStableWindows2016RDSOffice 安定版最新のWindows2016RDS(Office)パブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2016RDSOffice() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2016RDSOfficeTags, map[string]interface{}{
		"Name": "Windows Server 2016 for RDS(MS Office付)",
	})
}

// FindLatestStableWindows2016SQLServerWeb 安定版最新のWindows2016 SQLServer(Web) パブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2016SQLServerWeb() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2016SQLServerWeb, map[string]interface{}{
		"Name": "Windows Server 2016 for MS SQL 2016(Web)",
	})
}

// FindLatestStableWindows2016SQLServerStandard 安定版最新のWindows2016 SQLServer(Standard) パブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2016SQLServerStandard() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2016SQLServerStandard, map[string]interface{}{
		"Name": "Windows Server 2016 for MS SQL 2016(Standard)",
	})
}

// FindLatestStableWindows2016SQLServerStandardAll 安定版最新のWindows2016 SQLServer(RDS+Office) パブリックアーカイブを取得
func (api *ArchiveAPI) FindLatestStableWindows2016SQLServerStandardAll() (*sacloud.Archive, error) {
	return api.findByOSTags(archiveLatestStableWindows2016SQLServerStandard, map[string]interface{}{
		"Name": "Windows Server 2016 for MS SQL 2016(Std) with RDS / MS Office",
	})
}

// FindByOSType 指定のOS種別の安定版最新のパブリックアーカイブを取得
func (api *ArchiveAPI) FindByOSType(os ostype.ArchiveOSTypes) (*sacloud.Archive, error) {
	if f, ok := api.findFuncMapPerOSType[os]; ok {
		return f()
	}

	return nil, fmt.Errorf("OSType [%s] is invalid", os)
}

func (api *ArchiveAPI) findByOSTags(tags []string, filterMap ...map[string]interface{}) (*sacloud.Archive, error) {

	api.Reset().WithTags(tags)

	for _, filters := range filterMap {
		for key, filter := range filters {
			api.FilterMultiBy(key, filter)
		}
	}
	res, err := api.Find()
	if err != nil {
		return nil, fmt.Errorf("Archive [%s] error : %s", strings.Join(tags, ","), err)
	}

	if len(res.Archives) == 0 {
		return nil, fmt.Errorf("Archive [%s] Not Found", strings.Join(tags, ","))
	}

	return &res.Archives[0], nil

}
