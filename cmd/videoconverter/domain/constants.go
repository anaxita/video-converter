package domain

const EnvProd = "prod"
const EnvDebug = "debug"

// CacheURL a basic url for all cache video
const CacheURL = "https://cache-synergy.cdnvideo.ru/synergy/videoconverter/"

// channels for logging
const (
	ChDone = iota
	ChAll
	ChConverted
	ChNotConverted
	ChUploaded
	ChNotUploaded
)

// VQ video quality
type VQ int

const (
	Q360     VQ = 360
	Q480     VQ = 480
	Q720     VQ = 720
	Q1080    VQ = 1080
	QPreview VQ = 3333
)
