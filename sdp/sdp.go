package sdp
	"net"
)

type SessionDescription struct {
	Version string
	Timing  struct {
		StartTime uint64
		EndTime   uint64
	}
	Originator struct {
		Username string
	}
	Network struct {
		Family  int
		Address net.IPAddr
	}
	Media       []MediaDescription
	SessionName string
}

type MediaDescription struct {
	Type           string
	RTPPayloadType uint8
	SampleRate     uint32
	Port           uint16
	Encoding       string
}

func (s *SessionDescription) String() string {
	var output bytes.Buffer
	output.WriteString(buildVersionField(s.Version))
	output.WriteString(buildOriginatorField(s.Originator.Username, s.Timing.StartTime, s.Timing.EndTime, s.Network.Family, s.Network.Address))
	output.WriteString(fmt.Sprintf("s=%s\n", s.SessionName))
	output.WriteString(fmt.Sprintf("c=IN IP%d %s\n", s.Network.Family, s.Network.Address.IP.String()))
	output.WriteString(buildTransportFields(s.Media))
	output.WriteString(buildAttributeFields(s.Media))
	return output.String()
}

func buildVersionField(version string) string {
	return fmt.Sprintf("v=%s\n", version)
}

func buildOriginatorField(username string, startTime uint64, endTime uint64, family int, address net.IPAddr) string {
	return fmt.Sprintf("o=%s %d %d IN IP%d %s\n", username, startTime, endTime, family, address.IP.String())
}

func buildAttributeFields(mds []MediaDescription) string {
	var output bytes.Buffer
	for _, m := range mds {
		output.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d\n", m.RTPPayloadType, m.Encoding, m.SampleRate))
	}
	return output.String()
}

func buildTransportFields(mds []MediaDescription) string {
	var output bytes.Buffer
	for _, m := range mds {
		output.WriteString(fmt.Sprintf("m=%s %d RTP/AVP %d\n", m.Type, m.Port, m.RTPPayloadType))
	}
	return output.String()
}
