package ctx

func Create(context *Context) (err error) {
	return context.Save()
}

func Load(name string) (context *Context, err error) {
	context = &Context{
		Name: name,
	}

	err = context.Load()
	return
}

func Switch(context *Context, mode ContextMode) (err error) {
	cc := CurrentContext{
		Context: context,
		CtxMode: mode,
	}

	return cc.Save()
}

func Current() (context *Context, err error) {
	cc := CurrentContext{}
	if err = cc.Load(); err != nil {
		return
	}

	context = cc.Context
	return
}
