package conply

import (
	"os/user"
	"strings"
)

// Checks and prepare the environment.
func PrepareEnv(bundle string) error {
	paths := make(map[string]string, 3)
	paths["config"], _ = GetConfigDir(bundle)
	paths["cache"], _ = GetCacheDir(bundle)
	paths["dl"], _ = GetDlDir(bundle, "")
	for _, path := range paths {
		if !FileExists(path) {
			if err := Mkdir(path); err != nil {
				return err
			}
		}
	}

	return nil
}

// Returns absolute path to config directory.
func GetConfigDir(bundle string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return strings.Join([]string{usr.HomeDir, ".config", bundle}, PS), nil
}

// Get path to hotkeys config.
func GetHKPath(bundle string) (string, error) {
	path, err := GetConfigDir(bundle)
	return path + PS + "hotkeys.json", err
}

// Returns absolute path to cache directory.
func GetCacheDir(bundle string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return strings.Join([]string{usr.HomeDir, ".cache", bundle}, PS), nil
}

// Get path to channels cache storage.
func GetChannelsPath(bundle string) (string, error) {
	path, err := GetCacheDir(bundle)
	return path + PS + "channels.json", err
}

// Get path to channels cache storage with station key.
func GetChannelsPathWS(bundle, station string) (string, error) {
	path, err := GetCacheDir(bundle)
	return path + PS + station + ".json", err
}

// Returns absolute path to cache directory.
func GetDlDir(bundle, channel string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	chunks := []string{usr.HomeDir, "Music", bundle}
	if len(channel) > 0 {
		chunks = append(chunks, channel)
	}
	return strings.Join(chunks, PS), nil
}
