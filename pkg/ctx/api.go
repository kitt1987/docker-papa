package ctx

func Create(context *Context) (err error) {
	err = context.Save()
	return
}

func Switch(context *Context, mode ContextMode) (err error) {
	cc := CurrentContext{
		Context: context,
		CtxMode: mode,
	}

	return cc.Save()
}
