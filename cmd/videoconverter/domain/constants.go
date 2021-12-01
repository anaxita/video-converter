package domain

// VQ video quality
type VQ int

const (
	Q360     VQ = 360
	Q480     VQ = 480
	Q720     VQ = 720
	Q1080    VQ = 1080
	QPreview VQ = 3333
)

const CacheURL = "https://cache-synergy.cdnvideo.ru/synergy/"
const EnvProd = "prod"

const (
	ChDone = iota
	ChAll
	ChConverted
	ChNotConverted
	ChUploaded
	ChNotUploaded
)
