package main

import (
	"reflect"
	"strings"
	"time"

	"github.com/docker/go-units"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/projectatomic/libpod/cmd/podman/formats"
	"github.com/projectatomic/libpod/libpod"
	"github.com/projectatomic/libpod/pkg/inspect"
	"github.com/urfave/cli"
)

type imagesTemplateParams struct {
	Repository string
	Tag        string
	ID         string
	Digest     digest.Digest
	Created    string
	Size       string
}

type imagesJSONParams struct {
	ID      string        `json:"id"`
	Name    []string      `json:"names"`
	Digest  digest.Digest `json:"digest"`
	Created time.Time     `json:"created"`
	Size    *uint64       `json:"size"`
}

type imagesOptions struct {
	quiet        bool
	noHeading    bool
	noTrunc      bool
	digests      bool
	format       string
	outputformat string
}

var (
	imagesFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "display only image IDs",
		},
		cli.BoolFlag{
			Name:  "noheading, n",
			Usage: "do not print column headings",
		},
		cli.BoolFlag{
			Name:  "no-trunc, notruncate",
			Usage: "do not truncate output",
		},
		cli.BoolFlag{
			Name:  "digests",
			Usage: "show digests",
		},
		cli.StringFlag{
			Name:  "format",
			Usage: "Change the output format to JSON or a Go template",
		},
		cli.StringSliceFlag{
			Name:  "filter, f",
			Usage: "filter output based on conditions provided (default [])",
		},
	}

	imagesDescription = "lists locally stored images."
	imagesCommand     = cli.Command{
		Name:                   "images",
		Usage:                  "list images in local storage",
		Description:            imagesDescription,
		Flags:                  imagesFlags,
		Action:                 imagesCmd,
		ArgsUsage:              "",
		UseShortOptionHandling: true,
	}
)

func imagesCmd(c *cli.Context) error {
	if err := validateFlags(c, imagesFlags); err != nil {
		return err
	}

	runtime, err := getRuntime(c)
	if err != nil {
		return errors.Wrapf(err, "Could not get runtime")
	}
	defer runtime.Shutdown(false)
	var filterFuncs []libpod.ImageResultFilter
	var imageInput string
	if len(c.Args()) == 1 {
		imageInput = c.Args().Get(0)
	}

	if len(c.Args()) > 1 {
		return errors.New("'podman images' requires at most 1 argument")
	}

	if len(c.StringSlice("filter")) > 0 || len(strings.TrimSpace(imageInput)) != 0 {
		filterFuncs, err = CreateFilterFuncs(runtime, c, imageInput)
		if err != nil {
			return err
		}
	}

	opts := imagesOptions{
		quiet:     c.Bool("quiet"),
		noHeading: c.Bool("noheading"),
		noTrunc:   c.Bool("no-trunc"),
		digests:   c.Bool("digests"),
		format:    c.String("format"),
	}

	opts.outputformat = opts.setOutputFormat()
	/*
		podman does not implement --all for images

		intermediate images are only generated during the build process.  they are
		children to the image once built. until buildah supports caching builds,
		it will not generate these intermediate images.
	*/
	images, err := runtime.GetImageResults()
	if err != nil {
		return errors.Wrapf(err, "unable to get images")
	}

	var filteredImages []inspect.ImageResult
	// filter the images
	if len(c.StringSlice("filter")) > 0 || len(strings.TrimSpace(imageInput)) != 0 {
		filteredImages = libpod.FilterImages(images, filterFuncs)
	} else {
		filteredImages = images
	}

	return generateImagesOutput(runtime, filteredImages, opts)
}

func (i imagesOptions) setOutputFormat() string {
	if i.format != "" {
		// "\t" from the command line is not being recognized as a tab
		// replacing the string "\t" to a tab character if the user passes in "\t"
		return strings.Replace(i.format, `\t`, "\t", -1)
	}
	if i.quiet {
		return formats.IDString
	}
	format := "table {{.Repository}}\t{{.Tag}}\t"
	if i.noHeading {
		format = "{{.Repository}}\t{{.Tag}}\t"
	}
	if i.digests {
		format += "{{.Digest}}\t"
	}
	format += "{{.ID}}\t{{.Created}}\t{{.Size}}\t"
	return format
}

// imagesToGeneric creates an empty array of interfaces for output
func imagesToGeneric(templParams []imagesTemplateParams, JSONParams []imagesJSONParams) (genericParams []interface{}) {
	if len(templParams) > 0 {
		for _, v := range templParams {
			genericParams = append(genericParams, interface{}(v))
		}
		return
	}
	for _, v := range JSONParams {
		genericParams = append(genericParams, interface{}(v))
	}
	return
}

// getImagesTemplateOutput returns the images information to be printed in human readable format
func getImagesTemplateOutput(runtime *libpod.Runtime, images []inspect.ImageResult, opts imagesOptions) (imagesOutput []imagesTemplateParams) {
	for _, img := range images {
		createdTime := img.Created

		imageID := "sha256:" + img.ID
		if !opts.noTrunc {
			imageID = shortID(img.ID)
		}
		params := imagesTemplateParams{
			Repository: img.Repository,
			Tag:        img.Tag,
			ID:         imageID,
			Digest:     img.Digest,
			Created:    units.HumanDuration(time.Since((createdTime))) + " ago",
			Size:       units.HumanSizeWithPrecision(float64(*img.Size), 3),
		}
		imagesOutput = append(imagesOutput, params)
	}
	return
}

// getImagesJSONOutput returns the images information in its raw form
func getImagesJSONOutput(runtime *libpod.Runtime, images []inspect.ImageResult) (imagesOutput []imagesJSONParams) {
	for _, img := range images {
		params := imagesJSONParams{
			ID:      img.ID,
			Name:    img.RepoTags,
			Digest:  img.Digest,
			Created: img.Created,
			Size:    img.Size,
		}
		imagesOutput = append(imagesOutput, params)
	}
	return
}

// generateImagesOutput generates the images based on the format provided

func generateImagesOutput(runtime *libpod.Runtime, images []inspect.ImageResult, opts imagesOptions) error {
	if len(images) == 0 {
		return nil
	}
	var out formats.Writer

	switch opts.format {
	case formats.JSONString:
		imagesOutput := getImagesJSONOutput(runtime, images)
		out = formats.JSONStructArray{Output: imagesToGeneric([]imagesTemplateParams{}, imagesOutput)}
	default:
		imagesOutput := getImagesTemplateOutput(runtime, images, opts)
		out = formats.StdoutTemplateArray{Output: imagesToGeneric(imagesOutput, []imagesJSONParams{}), Template: opts.outputformat, Fields: imagesOutput[0].HeaderMap()}
	}
	return formats.Writer(out).Out()
}

// HeaderMap produces a generic map of "headers" based on a line
// of output
func (i *imagesTemplateParams) HeaderMap() map[string]string {
	v := reflect.Indirect(reflect.ValueOf(i))
	values := make(map[string]string)

	for i := 0; i < v.NumField(); i++ {
		key := v.Type().Field(i).Name
		value := key
		if value == "ID" {
			value = "Image" + value
		}
		values[key] = strings.ToUpper(splitCamelCase(value))
	}
	return values
}

// CreateFilterFuncs returns an array of filter functions based on the user inputs
// and is later used to filter images for output
func CreateFilterFuncs(r *libpod.Runtime, c *cli.Context, userInput string) ([]libpod.ImageResultFilter, error) {
	var filterFuncs []libpod.ImageResultFilter
	for _, filter := range c.StringSlice("filter") {
		splitFilter := strings.Split(filter, "=")
		switch splitFilter[0] {
		case "before":
			before := r.NewImage(splitFilter[1])
			_, beforeID, _ := before.GetLocalImageName()

			if before.LocalName == "" {
				return nil, errors.Errorf("unable to find image % in local stores", splitFilter[1])
			}
			img, err := r.GetImage(beforeID)
			if err != nil {
				return nil, err
			}
			filterFuncs = append(filterFuncs, libpod.ImageCreatedBefore(img.Created))
		case "after":
			after := r.NewImage(splitFilter[1])
			_, afterID, _ := after.GetLocalImageName()

			if after.LocalName == "" {
				return nil, errors.Errorf("unable to find image % in local stores", splitFilter[1])
			}
			img, err := r.GetImage(afterID)
			if err != nil {
				return nil, err
			}
			filterFuncs = append(filterFuncs, libpod.ImageCreatedAfter(img.Created))
		case "dangling":
			filterFuncs = append(filterFuncs, libpod.ImageDangling())
		case "label":
			labelFilter := strings.Join(splitFilter[1:], "=")
			filterFuncs = append(filterFuncs, libpod.ImageLabel(labelFilter))
		default:
			return nil, errors.Errorf("invalid filter %s ", splitFilter[0])
		}
	}
	if len(strings.TrimSpace(userInput)) != 0 {
		filterFuncs = append(filterFuncs, libpod.OutputImageFilter(userInput))
	}
	return filterFuncs, nil
}
