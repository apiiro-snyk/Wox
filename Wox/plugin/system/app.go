package system

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"wox/plugin"
	"wox/util"
)

func init() {
	plugin.AllSystemPlugin = append(plugin.AllSystemPlugin, &AppPlugin{})
}

type AppPlugin struct {
	api  plugin.API
	apps []appInfo
}

type appInfo struct {
	Name string
	Path string
}

func (i *AppPlugin) GetMetadata() plugin.Metadata {
	return plugin.Metadata{
		Id:            "ea2b6859-14bc-4c89-9c88-627da7379141",
		Name:          "App",
		Author:        "Wox Launcher",
		Version:       "1.0.0",
		MinWoxVersion: "2.0.0",
		Runtime:       "Nodejs",
		Description:   "Search app installed on your computer",
		Icon:          "",
		Entry:         "",
		TriggerKeywords: []string{
			"*",
		},
		SupportedOS: []string{
			"Windows",
			"Macos",
			"Linux",
		},
	}
}

func (i *AppPlugin) Init(ctx context.Context, initParams plugin.InitParams) {
	i.api = initParams.API

	appCache, cacheErr := i.loadAppCache(ctx)
	if cacheErr == nil {
		i.apps = appCache
	}

	util.Go(ctx, "index apps", func() {
		i.indexApps(util.NewTraceContext())
	})
}

func (i *AppPlugin) Query(ctx context.Context, query plugin.Query) []plugin.QueryResult {
	var results []plugin.QueryResult
	for _, info := range i.apps {
		if strings.Contains(strings.ToLower(info.Name), strings.ToLower(query.Search)) {
			results = append(results, plugin.QueryResult{
				Id:       uuid.NewString(),
				Title:    info.Name,
				SubTitle: info.Path,
				Icon:     plugin.WoxImage{},
				Action: func() {
				},
			})
		}
	}

	return results
}

func (i *AppPlugin) indexApps(ctx context.Context) {
	startTimestamp := util.GetSystemTimestamp()
	var apps []appInfo
	if runtime.GOOS == "darwin" {
		apps = i.getMacApps(ctx)
	}

	if len(apps) > 0 {
		i.api.Log(ctx, fmt.Sprintf("indexed %d apps", len(i.apps)))
		i.apps = apps

		var cachePath = i.getAppCachePath()
		cacheContent, marshalErr := json.Marshal(apps)
		if marshalErr != nil {
			i.api.Log(ctx, fmt.Sprintf("error marshalling app cache: %s", marshalErr.Error()))
			return
		}
		writeErr := os.WriteFile(cachePath, cacheContent, 0644)
		if writeErr != nil {
			i.api.Log(ctx, fmt.Sprintf("error writing app cache: %s", writeErr.Error()))
			return
		}
		i.api.Log(ctx, fmt.Sprintf("wrote app cache to %s", cachePath))
	}

	i.api.Log(ctx, fmt.Sprintf("indexed %d apps, cost %d ms", len(i.apps), util.GetSystemTimestamp()-startTimestamp))
}

func (i *AppPlugin) getAppCachePath() string {
	return path.Join(os.TempDir(), "wox-app-cache.json")
}

func (i *AppPlugin) loadAppCache(ctx context.Context) ([]appInfo, error) {
	startTimestamp := util.GetSystemTimestamp()
	i.api.Log(ctx, "start to load app cache")
	var cachePath = i.getAppCachePath()
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		i.api.Log(ctx, "app cache file not found")
		return nil, err
	}

	cacheContent, readErr := os.ReadFile(cachePath)
	if readErr != nil {
		i.api.Log(ctx, fmt.Sprintf("error reading app cache file: %s", readErr.Error()))
		return nil, readErr
	}

	var apps []appInfo
	unmarshalErr := json.Unmarshal(cacheContent, &apps)
	if unmarshalErr != nil {
		i.api.Log(ctx, fmt.Sprintf("error unmarshalling app cache file: %s", unmarshalErr.Error()))
		return nil, unmarshalErr
	}

	i.api.Log(ctx, fmt.Sprintf("loaded %d apps from cache, cost %d ms", len(apps), util.GetSystemTimestamp()-startTimestamp))
	return apps, nil
}

func (i *AppPlugin) getMacApps(ctx context.Context) []appInfo {
	i.api.Log(ctx, "start to get mac apps")

	var appDirectories = []string{
		"/Applications",
		"/Applications/Utilities",
		"/System/Applications",
		"/System/Library/PreferencePanes",
	}

	var appDirectoryPaths []string
	for _, appDirectory := range appDirectories {
		// get all .app directories in appDirectory
		appDir, readErr := os.ReadDir(appDirectory)
		if readErr != nil {
			i.api.Log(ctx, fmt.Sprintf("error reading directory %s: %s", appDirectory, readErr.Error()))
			continue
		}

		for _, entry := range appDir {
			if strings.HasSuffix(entry.Name(), ".app") || strings.HasSuffix(entry.Name(), ".prefPane") {
				appDirectoryPaths = append(appDirectoryPaths, path.Join(appDirectory, entry.Name()))
			}
		}
	}

	var appInfos []appInfo
	for _, directoryPath := range appDirectoryPaths {
		info, getErr := i.getMacAppInfo(ctx, directoryPath)
		if getErr != nil {
			i.api.Log(ctx, fmt.Sprintf("error getting app info for %s: %s", directoryPath, getErr.Error()))
			continue
		}

		appInfos = append(appInfos, info)
	}

	return appInfos
}

func (i *AppPlugin) getMacAppInfo(ctx context.Context, path string) (appInfo, error) {
	out, err := exec.Command("mdls", "-name", "kMDItemDisplayName", "-raw", path).Output()
	if err != nil {
		return appInfo{}, fmt.Errorf("failed to get app name: %w", err)
	}

	return appInfo{
		Name: strings.TrimSpace(string(out)),
		Path: path,
	}, nil
}
