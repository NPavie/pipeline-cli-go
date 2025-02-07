package cli

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/daisy/pipeline-clientlib-go"
	"golang.org/x/exp/slices"
)

const (
	MSG_WAIT = 1000 * time.Millisecond //waiting time for getting messages
)

// Convinience for testing, propably move to pipeline-clientlib-go
type PipelineApi interface {
	SetCredentials(string, string)
	SetUrl(string)
	Alive() (alive pipeline.Alive, err error)
	Scripts() (scripts pipeline.Scripts, err error)
	Script(id string) (script pipeline.Script, err error)
	JobRequest(newJob pipeline.JobRequest, data []byte) (job pipeline.Job, err error)
	StylesheetParametersRequest(newReq pipeline.StylesheetParametersRequest, data []byte) (params pipeline.StylesheetParameters, err error)
	ScriptUrl(id string) string
	Job(string, int) (pipeline.Job, error)
	DeleteJob(id string) (bool, error)
	Results(id string, w io.Writer) (bool, error)
	Log(id string) ([]byte, error)
	Jobs() (pipeline.Jobs, error)
	Halt(key string) error
	Clients() (clients []pipeline.Client, err error)
	NewClient(in pipeline.Client) (out pipeline.Client, err error)
	ModifyClient(in pipeline.Client, id string) (out pipeline.Client, err error)
	DeleteClient(id string) (ok bool, err error)
	Client(id string) (out pipeline.Client, err error)
	Properties() (props []pipeline.Property, err error)
	Sizes() (sizes pipeline.JobSizes, err error)
	Queue() ([]pipeline.QueueJob, error)
	MoveUp(id string) ([]pipeline.QueueJob, error)
	MoveDown(id string) ([]pipeline.QueueJob, error)
}

// Maintains some information about the pipeline client
type PipelineLink struct {
	pipeline       PipelineApi //Allows access to the pipeline fwk
	config         Config
	Version        string //Framework version
	Authentication bool   //Framework authentication
	FsAllow        bool   //Framework mode
}

func NewLink(conf Config) (pLink *PipelineLink) {

	pLink = &PipelineLink{
		pipeline: pipeline.NewPipeline(conf.Url()),
		config:   conf,
	}
	//assure that the pipeline is up

	return
}

func (p *PipelineLink) Init() error {
	log.Println("Initialising link")
	p.pipeline.SetUrl(p.config.Url())
	if err := bringUp(p); err != nil {
		return err
	}
	//set the credentials
	if p.Authentication {
		if !(len(p.config[CLIENTKEY].(string)) > 0 && len(p.config[CLIENTSECRET].(string)) > 0) {
			return errors.New("link: Authentication required but client_key and client_secret are not set. Please, check the configuration")
		}
		p.pipeline.SetCredentials(p.config[CLIENTKEY].(string), p.config[CLIENTSECRET].(string))
	}
	return nil
}
func (p PipelineLink) IsLocal() bool {
	return p.FsAllow
}

// checks if the pipeline is up
// otherwise it brings it up and fills the
// link object
func bringUp(pLink *PipelineLink) error {
	alive, err := pLink.pipeline.Alive()
	if err != nil {
		// Requested behavior :
		// dp2 launches pipeline-ui (if needed); dp2 calls pipeline-ui as cli handler; pipeline-ui calls dp2 with the port info as an argument

		// When port attribute is set, the cli tool should not try to start a pipeline instance
		// and directly connect to pipeline instance for which the port is set on the command line
		if pLink.config[STARTING].(bool) && !(slices.Contains(os.Args[1:], "--port") || slices.Contains(os.Args[1:], "--host")) {
			execpath := pLink.config.ExecPath()
			// if execpath ends with DAISY Pipeline or DAISY Pipeline.exe
			// we forward the command to the electron app
			if strings.HasSuffix(execpath, "DAISY Pipeline") || strings.HasSuffix(execpath, "DAISY Pipeline.exe") {
				args := os.Args[1:]
				if len(os.Args) == 1 {
					args = append(args, "help")
				}
				absPath, err := exec.LookPath(execpath)
				if err != nil {
					absPath = execpath
				}
				cmd := exec.Command(absPath, args...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("Could not use the pipeline app: %v", err)
				} else {
					os.Exit(cmd.ProcessState.ExitCode())
				}
			} else {
				alive, err = NewPipelineLauncher(
					pLink.pipeline,
					execpath,
					10,
				).Launch(os.Stdout)
				if err != nil {
					return fmt.Errorf("Error bringing the pipeline2 up %v", err.Error())
				}
			}
		} else {
			return fmt.Errorf("Could not connect to the webservice and I'm not configured to start one\n\tError: %v", err.Error())
		}
	}
	log.Println("Setting values")
	pLink.Version = alive.Version
	pLink.FsAllow = alive.FsAllow
	pLink.Authentication = alive.Authentication
	return nil
}

// ScriptList returns the list of scripts available in the framework
func (p PipelineLink) Scripts() (scripts []pipeline.Script, err error) {
	scriptsStruct, err := p.pipeline.Scripts()
	if err != nil {
		return
	}
	scripts = make([]pipeline.Script, len(scriptsStruct.Scripts))
	//fill the script list with the complete definition
	for idx, script := range scriptsStruct.Scripts {
		scripts[idx], err = p.pipeline.Script(script.Id)
		if err != nil {
			err = fmt.Errorf("Error loading script %v: %v", script.Id, err)
			return nil, err
		}
	}
	return scripts, err
}

// Gets the job identified by the jobId
func (p PipelineLink) Job(jobId string) (job pipeline.Job, err error) {
	job, err = p.pipeline.Job(jobId, 0)
	return
}

// Deletes the given job
func (p PipelineLink) Delete(jobId string) (ok bool, err error) {
	ok, err = p.pipeline.DeleteJob(jobId)
	return
}

// Return the zipped results as a []byte
func (p PipelineLink) Results(jobId string, w io.Writer) (ok bool, err error) {
	return p.pipeline.Results(jobId, w)
}
func (p PipelineLink) Log(jobId string) (data []byte, err error) {
	data, err = p.pipeline.Log(jobId)
	return
}
func (p PipelineLink) Jobs() (jobs []pipeline.Job, err error) {
	pJobs, err := p.pipeline.Jobs()
	if err != nil {
		return
	}
	jobs = pJobs.Jobs
	return
}

// Admin
func (p PipelineLink) Halt(key string) error {
	return p.pipeline.Halt(key)
}

func (p PipelineLink) Clients() (clients []pipeline.Client, err error) {
	return p.pipeline.Clients()
}

func (p PipelineLink) NewClient(newClient pipeline.Client) (client pipeline.Client, err error) {
	return p.pipeline.NewClient(newClient)
}
func (p PipelineLink) DeleteClient(id string) (ok bool, err error) {
	return p.pipeline.DeleteClient(id)
}
func (p PipelineLink) Client(id string) (out pipeline.Client, err error) {
	return p.pipeline.Client(id)
}

func (p PipelineLink) ModifyClient(data pipeline.Client, id string) (client pipeline.Client, err error) {
	return p.pipeline.ModifyClient(data, id)
}
func (p PipelineLink) Properties() (props []pipeline.Property, err error) {
	return p.pipeline.Properties()
}
func (p PipelineLink) Sizes() (sizes pipeline.JobSizes, err error) {
	return p.pipeline.Sizes()
}

func (p PipelineLink) Queue() (queue []pipeline.QueueJob, err error) {
	return p.pipeline.Queue()
}
func (p PipelineLink) MoveUp(id string) (queue []pipeline.QueueJob, err error) {
	return p.pipeline.MoveUp(id)
}
func (p PipelineLink) MoveDown(id string) (queue []pipeline.QueueJob, err error) {
	return p.pipeline.MoveDown(id)
}

// Convience structure to handle message and errors from the communication with the pipelineApi
type Message struct {
	Message  string
	Level    string
	Depth    int
	Status   string
	Progress float64
	Error    error
}

// Returns a simple string representation of the messages strucutre:
// [LEVEL]   Message content
func (m Message) String() string {
	if m.Message != "" {
		indent := ""
		for i := 1; i <= m.Depth; i++ {
			indent += "  "
		}
		level := "[" + m.Level + "]"
		for len(level) < 10 {
			level += " "
		}
		str := ""
		for i, line := range regexp.MustCompile("\r?\n|\r").Split(m.Message, -1) {
			if i == 0 {
				str += fmt.Sprintf("%v %v%v", level, indent, line)
			} else {
				str += fmt.Sprintf("\n           %v%v", indent, line)
			}
		}
		return str
	} else {
		return ""
	}
}

// Executes the job request and returns a channel fed with the job's messages,errors, and status.
// The last message will have no contents but the status of the in which the job finished
func (p PipelineLink) Execute(jobReq JobRequest) (job pipeline.Job, messages chan Message, err error) {
	req, err := jobRequestToPipeline(jobReq, p)
	if err != nil {
		return
	}
	log.Printf("data len exec %v", len(jobReq.Data))
	job, err = p.pipeline.JobRequest(req, jobReq.Data)
	if err != nil {
		return
	}
	messages = make(chan Message)
	if !jobReq.Background {
		go getAsyncMessages(p, job.Id, messages)
	} else {
		close(messages)
	}
	return
}

func (p PipelineLink) StylesheetParameters(paramReq StylesheetParametersRequest) (params pipeline.StylesheetParameters, err error) {
	req := pipeline.StylesheetParametersRequest{
		Media:               pipeline.Media{Value: paramReq.Medium},
		UserAgentStylesheet: pipeline.UserAgentStylesheet{Mediatype: paramReq.ContentType},
	}
	log.Printf("data len exec %v", len(paramReq.Data))
	params, err = p.pipeline.StylesheetParametersRequest(req, paramReq.Data)
	return
}

// Feeds the channel with the messages describing the job's execution
func getAsyncMessages(p PipelineLink, jobId string, messages chan Message) {
	msgSeq := -1
	for {
		job, err := p.pipeline.Job(jobId, msgSeq)
		if err != nil {
			messages <- Message{Error: err}
			close(messages)
			return
		}
		n := msgSeq
		if len(job.Messages.Message) > 0 {
			n = flattenMessages(job.Messages.Message, messages, job.Status, job.Messages.Progress, msgSeq+1, 0)
		}
		if n > msgSeq {
			msgSeq = n
		} else {
			messages <- Message{Progress: job.Messages.Progress}
		}
		if job.Status == "SUCCESS" || job.Status == "ERROR" || job.Status == "FAIL" {
			messages <- Message{Status: job.Status}
			close(messages)
			return
		}
		time.Sleep(MSG_WAIT)
	}

}

// Flatten message coming from the Pipeline job and feed them into the channel
// Return the sequence number of the inner message with the highest sequence number
func flattenMessages(from []pipeline.Message, to chan Message, status string, progress float64, firstSeq int, depth int) (lastSeq int) {
	lastSeq = -1
	for _, msg := range from {
		seq := msg.Sequence
		if seq >= firstSeq {
			to <- Message{Message: msg.Content, Level: msg.Level, Depth: depth, Status: status, Progress: progress}
			if seq > lastSeq {
				lastSeq = seq
			}
		}
		if len(msg.Message) > 0 {
			seq := flattenMessages(msg.Message, to, status, progress, firstSeq, depth+1)
			if seq > lastSeq {
				lastSeq = seq
			}
		}
	}
	return lastSeq
}

func jobRequestToPipeline(req JobRequest, p PipelineLink) (pReq pipeline.JobRequest, err error) {
	href := p.pipeline.ScriptUrl(req.Script)
	pReq = pipeline.JobRequest{
		Script:   pipeline.Script{Href: href},
		Nicename: req.Nicename,
		Priority: req.Priority,
	}
	for name, values := range req.Inputs {
		input := pipeline.Input{Name: name}
		for _, value := range values {
			input.Items = append(input.Items, pipeline.Item{Value: value.String()})
		}
		pReq.Inputs = append(pReq.Inputs, input)
	}
	var stylesheetParametersOption pipeline.Option
	for name, values := range req.Options {
		option := pipeline.Option{Name: name}
		if len(values) > 1 {
			for _, value := range values {
				option.Items = append(option.Items, pipeline.Item{Value: value})
			}
		} else {
			option.Value = values[0]
		}
		if name == "stylesheet-parameters" {
			stylesheetParametersOption = option
		} else {
			pReq.Options = append(pReq.Options, option)
		}
	}
	var params []string
	for name, param := range req.StylesheetParameters {
		switch param.Type.(type) {
		case pipeline.XsBoolean,
			pipeline.XsInteger,
			pipeline.XsNonNegativeInteger:
			params = append(params, fmt.Sprintf("%s: %s", name, param.Value))
		default:
			params = append(params,
				fmt.Sprintf("%s: '%s'", name, strings.NewReplacer(
					"\n", "\\A ",
					"'", "\\27 ",
				).Replace(param.Value)))
		}
	}
	if len(params) > 0 {
		value := fmt.Sprintf("(%s)", strings.Join(params, ", "))
		if stylesheetParametersOption.Name == "" {
			stylesheetParametersOption = pipeline.Option{Name: "stylesheet-parameters"}
			stylesheetParametersOption.Value = value
		} else {
			if stylesheetParametersOption.Value != "" {
				stylesheetParametersOption.Items = append(
					stylesheetParametersOption.Items,
					pipeline.Item{Value: stylesheetParametersOption.Value})
				stylesheetParametersOption.Value = ""
			}
			stylesheetParametersOption.Items = append(
				stylesheetParametersOption.Items,
				pipeline.Item{Value: value})
		}
	}
	if stylesheetParametersOption.Name != "" {
		pReq.Options = append(pReq.Options, stylesheetParametersOption)
	}
	return
}
