package shared_model

// InfoPlist represents the top-level structure of an Info.plist file.
type InfoPlist struct {
	CFBundleDevelopmentRegion            string                `json:"CFBundleDevelopmentRegion"`
	CFBundleExecutable                   string                `json:"CFBundleExecutable"`
	CFBundleIdentifier                   string                `json:"CFBundleIdentifier"`
	CFBundleInfoDictionaryVersion        string                `json:"CFBundleInfoDictionaryVersion"`
	CFBundleName                         string                `json:"CFBundleName"`
	CFBundlePackageType                  string                `json:"CFBundlePackageType"`
	CFBundleShortVersionString           string                `json:"CFBundleShortVersionString"`
	CFBundleSignature                    string                `json:"CFBundleSignature"`
	CFBundleVersion                      string                `json:"CFBundleVersion"`
	LSRequiresIPhoneOS                   bool                  `json:"LSRequiresIPhoneOS"`
	UILaunchStoryboardName               string                `json:"UILaunchStoryboardName"`
	UIRequiredDeviceCapabilities         []string              `json:"UIRequiredDeviceCapabilities"`
	UISupportedInterfaceOrientations     []string              `json:"UISupportedInterfaceOrientations"`
	NSCameraUsageDescription             string                `json:"NSCameraUsageDescription"`
	NSLocationWhenInUseUsageDescription  string                `json:"NSLocationWhenInUseUsageDescription"`
	NSAppTransportSecurity               *AppTransportSecurity `json:"NSAppTransportSecurity"` // Pointer to nested struct
	UIUserInterfaceStyle                 string                `json:"UIUserInterfaceStyle"`
	UISupportedInterfaceOrientationsIpad []string              `json:"UISupportedInterfaceOrientations~ipad"` // Note the '~ipad' in the JSON key
}

// AppTransportSecurity represents the nested dictionary for NSAppTransportSecurity.
type AppTransportSecurity struct {
	NSAllowsArbitraryLoads bool `json:"NSAllowsArbitraryLoads"`
}
