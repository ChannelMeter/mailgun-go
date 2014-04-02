package mailgun

import (
	"encoding/json"
	"errors"
	"github.com/mbanzon/simplehttp"
	"time"
)

// Message structures contain both the message text and the envelop for an e-mail message.
// At this time, please note that a message may NOT have file attachments.
type Message struct {
	from         string
	to           []string
	cc           []string
	bcc          []string
	subject      string
	text         string
	html         string
	tags         []string
	campaigns    []string
	dkim         bool
	deliveryTime *time.Time
	attachments  []string
	inlines      []string

	testMode       bool
	tracking       bool
	trackingClicks bool
	trackingOpens  bool
	headers        map[string]string
	variables      map[string]string

	dkimSet           bool
	trackingSet       bool
	trackingClicksSet bool
	trackingOpensSet  bool
}

type sendMessageResponse struct {
	Message string `json:"message"`
	Id      string `json:"id"`
}

// NewMessage returns a new e-mail message with the simplest envelop needed to send.
func NewMessage(from string, subject string, text string, to ...string) *Message {
	return &Message{from: from, subject: subject, text: text, to: to}
}

func (m *Message) AddAttachment(attachment string) {
	m.attachments = append(m.attachments, attachment)
}

func (m *Message) AddInline(inline string) {
	m.inlines = append(m.inlines, inline)
}

func (m *Message) AddRecipient(recipient string) {
	m.to = append(m.to, recipient)
}

func (m *Message) AddCC(recipient string) {
	m.cc = append(m.cc, recipient)
}

func (m *Message) AddBCC(recipient string) {
	m.bcc = append(m.bcc, recipient)
}

func (m *Message) SetHtml(html string) {
	m.html = html
}

// AddTag attaches a tag to the message.  Tags are useful for metrics gathering and event tracking purposes.
// Refer to the Mailgun documentation for further details.
func (m *Message) AddTag(tag string) {
	m.tags = append(m.tags, tag)
}

func (m *Message) AddCampaign(campaign string) {
	m.campaigns = append(m.campaigns, campaign)
}

func (m *Message) SetDKIM(dkim bool) {
	m.dkim = dkim
	m.dkimSet = true
}

func (m *Message) EnableTestMode() {
	m.testMode = true
}

// SetDeliveryTime schedules the message for transmission at the indicated time.
// Pass nil to remove any installed schedule.
func (m *Message) SetDeliveryTime(dt time.Time) {
	pdt := new(time.Time)
	*pdt = dt
	m.deliveryTime = pdt
}

// SetTracking sets the o:tracking message parameter to adjust, on a message-by-message basis, whether or not Mailgun will rewrite URLs to facilitate event tracking,
// such as opens, clicks, unsubscribes, etc.  Note: simply calling this method ensures that the o:tracking header is passed in with the message.  Its yes/no setting
// is determined by the call's parameter.  Note that this header is not passed on to the final recipient(s).
func (m *Message) SetTracking(tracking bool) {
	m.tracking = tracking
	m.trackingSet = true
}

func (m *Message) SetTrackingClicks(trackingClicks bool) {
	m.trackingClicks = trackingClicks
	m.trackingClicksSet = true
}

func (m *Message) SetTrackingOpens(trackingOpens bool) {
	m.trackingOpens = trackingOpens
	m.trackingOpensSet = true
}

func (m *Message) AddHeader(header, value string) {
	if m.headers == nil {
		m.headers = make(map[string]string)
	}
	m.headers[header] = value
}

func (m *Message) AddVariable(variable string, value interface{}) error {
	j, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if m.variables == nil {
		m.variables = make(map[string]string)
	}
	m.variables[variable] = string(j)
	return nil
}

// Send attempts to queue a message (see Message, NewMessage, and its methods) for delivery.
// It returns the Mailgun server response, which consists of two components:
// a human-readable status message, and a message ID.  The status and message ID are set only
// if no error occurred.
func (m *mailgunImpl) Send(message *Message) (mes string, id string, err error) {
	if !message.validateMessage() {
		err = errors.New("Message not valid")
	} else {
		r := simplehttp.NewHTTPRequest(generateApiUrl(m, messagesEndpoint))

		payload := simplehttp.NewFormDataPayload()

		payload.AddValue("from", message.from)
		payload.AddValue("subject", message.subject)
		payload.AddValue("text", message.text)
		for _, to := range message.to {
			payload.AddValue("to", to)
		}
		for _, cc := range message.cc {
			payload.AddValue("cc", cc)
		}
		for _, bcc := range message.bcc {
			payload.AddValue("bcc", bcc)
		}
		for _, tag := range message.tags {
			payload.AddValue("o:tag", tag)
		}
		for _, campaign := range message.campaigns {
			payload.AddValue("o:campaign", campaign)
		}
		if message.html != "" {
			payload.AddValue("html", message.html)
		}
		if message.dkimSet {
			payload.AddValue("o:dkim", yesNo(message.dkim))
		}
		if message.deliveryTime != nil {
			payload.AddValue("o:deliverytime", message.deliveryTime.Format("Mon, 2 Jan 2006 15:04:05 MST"))
		}
		if message.testMode {
			payload.AddValue("o:testmode", "yes")
		}
		if message.trackingSet {
			payload.AddValue("o:tracking", yesNo(message.tracking))
		}
		if message.trackingClicksSet {
			payload.AddValue("o:tracking-clicks", yesNo(message.trackingClicks))
		}
		if message.trackingOpensSet {
			payload.AddValue("o:tracking-opens", yesNo(message.trackingOpens))
		}
		if message.headers != nil {
			for header, value := range message.headers {
				payload.AddValue("h:"+header, value)
			}
		}
		if message.variables != nil {
			for variable, value := range message.variables {
				payload.AddValue("v:"+variable, value)
			}
		}
		if message.attachments != nil {
			for _, attachment := range message.attachments {
				payload.AddFile("attachment", attachment)
			}
		}
		if message.inlines != nil {
			for _, inline := range message.inlines {
				payload.AddFile("inline", inline)
			}
		}
		r.SetBasicAuth(basicAuthUser, m.ApiKey())

		var response sendMessageResponse
		_, err = r.PostResponseFromJSON(payload, &response)
		if err == nil {
			mes = response.Message
			id = response.Id
		}
	}

	return
}

// yesNo translates a true/false boolean value into a yes/no setting suitable for the Mailgun API.
func yesNo(b bool) string {
	if b {
		return "yes"
	} else {
		return "no"
	}
}

// validateMessage returns true if, and only if,
// a Message instance is sufficiently initialized to send via the Mailgun interface.
func (m *Message) validateMessage() bool {
	if m == nil {
		return false
	}

	if m.from == "" {
		return false
	}

	if !validateStringList(m.to, true) {
		return false
	}

	if !validateStringList(m.cc, false) {
		return false
	}

	if !validateStringList(m.bcc, false) {
		return false
	}

	if !validateStringList(m.tags, false) {
		return false
	}

	if !validateStringList(m.campaigns, false) || len(m.campaigns) > 3 {
		return false
	}

	if m.text == "" {
		return false
	}

	return true
}

// validateStringList returns true if, and only if,
// a slice of strings exists AND all of its elements exist,
// OR if the slice doesn't exist AND it's not required to exist.
// The requireOne parameter indicates whether the list is required to exist.
func validateStringList(list []string, requireOne bool) bool {
	hasOne := false

	if list == nil {
		return !requireOne
	} else {
		for _, a := range list {
			if a == "" {
				return false
			} else {
				hasOne = hasOne || true
			}
		}
	}

	return hasOne
}
