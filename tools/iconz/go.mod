// Throwaway module: rasterizes assets/vigil.svg to PNGs for the brand assets and
// the Windows binary icon. Kept separate so the main module's go.mod stays clean.
module vigil-iconz

go 1.25.0

require (
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef
	github.com/tc-hib/winres v0.3.1
)

require (
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	golang.org/x/image v0.12.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)
