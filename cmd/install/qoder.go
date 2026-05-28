package main

func generateQoderAdapters(gpowersHome string) error {
	return generateFlatAdapters(gpowersHome, "qoder")
}

func registerQoder(gpowersHome string) error {
	return registerFlatSkills(gpowersHome, "qoder")
}
