package utils

type Config struct {
	Debug					bool		`json:"debug"`
	BindAddressList			[]string	`json:"bind_address_list"`
	FileBasePath			string		`json:"file_base_path"`
	MsqlConnectionString	string		`json:"msql_connection_string"`
	QueueInitSize			int			`json:"queue_init_size"`
	RecoveryDirPath			string		`json:"recovery_dir_path"`
	RecoveryFileSize		int			`json:"recovery_file_size"`
	IpWhiteList				[]string	`json:"ip_white_list"`
	GzipEnable				bool		`json:"gzip_enable"`
}