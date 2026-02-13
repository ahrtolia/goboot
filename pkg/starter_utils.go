package app

func enabledByConfig(ctx *Context, key string, section string, defaultVal bool) bool {
	if ctx == nil || ctx.Config == nil {
		return defaultVal
	}
	v := ctx.Config.GetViper()
	if v == nil {
		return defaultVal
	}
	if key != "" && v.InConfig(key) {
		return v.GetBool(key)
	}
	if section != "" && v.InConfig(section) {
		return true
	}
	return defaultVal
}
