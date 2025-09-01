package image_loader

//// Usage Example
//func main() {
//
//	// Create cache (1000 items capacity)
//	cache, _ := lru.New[string, []byte](1000)
//
//	// Initialize loader (with local image directory)
//	loader := NewImageLoader(cache, "")
//
//	// Load various image types
//	images := []string{
//		//"/var/cloud/upload/upload5/20190809_000407.jpg",
//		//"Screenshot_20240113_180718_Instagram.jpg",
//		//"Screenshot_20240120_020041_Instagram.jpg",
//		"/media/mahdi/Cloud/apps/Photos/mahdi_abdolmaleki/assets/18.jpg",
//		"/media/mahdi/Cloud/apps/Photos/mahdi_abdolmaleki/assets/17.jpg",
//		"/media/mahdi/Cloud/apps/Photos/mahdi_abdolmaleki/assets/25.jpg",
//		"/media/mahdi/Cloud/apps/Photos/mahdi_abdolmaleki/assets/26.jpg",
//		"/media/mahdi/Cloud/apps/Photos/mahdi_abdolmaleki/assets/27.jpg",
//		"/media/mahdi/Cloud/apps/Photos/mahdi_abdolmaleki/assets/28.jpg",
//
//		//"https://mahdiali.s3.ir-thr-at1.arvanstorage.ir/%D9%86%D9%82%D8%B4%D9%87-%D8%AA%D8%A7%DB%8C%D9%85%D8%B1-%D8%B1%D8%A7%D9%87-%D9%BE%D9%84%D9%87-%D8%B3%D9%87-%D8%B3%DB%8C%D9%85.jpg?versionId=", // Network URL
//		//"https://mahdicpp.s3.ir-thr-at1.arvanstorage.ir/0f470b87c13e25bc4211683711e71e2a.jpg?versionId=",
//	}
//
//	ctx := context.Background()
//	for _, img := range images {
//		data, err := loader.LoadImage(ctx, img)
//		if err != nil {
//			log.Printf("Failed to load %s: %v", img, err)
//			continue
//		}
//		fmt.Printf("Loaded %s (%d kB)\n", img, len(data)/1024)
//	}
//
//	// Print metrics
//	f, n, g, e, avg := loader.Metrics()
//	fmt.Printf("\nLoader Metrics:\n")
//	fmt.Printf("File loads: %d\n", f)
//	fmt.Printf("Network loads: %d\n", n)
//	fmt.Printf("Generated images: %d\n", g)
//	fmt.Printf("Errors: %d\n", e)
//	fmt.Printf("Avg load time: %s\n", avg)
//}
