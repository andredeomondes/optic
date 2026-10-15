package atvremote

import "google.golang.org/protobuf/encoding/protowire"

// Polo pairing messages (port 6467), hand-encoded from the community-documented
// polo.proto. Every message is an OuterMessage carrying exactly one payload.
//
//	message OuterMessage {
//	  uint32 protocol_version = 1;   // 2
//	  Status status           = 2;   // 200 OK, 400 error, 401 bad config, 402 bad secret
//	  PairingRequest    pairing_request     = 10;
//	  PairingRequestAck pairing_request_ack = 11;
//	  Options           options             = 20;
//	  Configuration     configuration       = 30;
//	  ConfigurationAck  configuration_ack   = 31;
//	  Secret            secret              = 40;
//	  SecretAck         secret_ack          = 41;
//	}
const (
	fieldPairingRequest    protowire.Number = 10
	fieldPairingRequestAck protowire.Number = 11
	fieldOptions           protowire.Number = 20
	fieldConfiguration     protowire.Number = 30
	fieldConfigurationAck  protowire.Number = 31
	fieldSecret            protowire.Number = 40
	fieldSecretAck         protowire.Number = 41
)

const (
	poloProtocolVersion = 2
	statusOK            = 200
	statusBadSecret     = 402

	encodingHexadecimal = 3 // Options.Encoding.EncodingType
	roleInput           = 1 // Options.RoleType: we type the code shown by the TV
	codeSymbols         = 6
)

var poloStatusText = map[uint64]string{
	400: "erro genérico",
	401: "configuração recusada",
	402: "código incorreto",
}

func outer(field protowire.Number, payload []byte) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, poloProtocolVersion)
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, statusOK)
	b = protowire.AppendTag(b, field, protowire.BytesType)
	b = protowire.AppendBytes(b, payload)
	return b
}

func appendString(b []byte, num protowire.Number, s string) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendString(b, s)
}

func appendUint(b []byte, num protowire.Number, v uint64) []byte {
	b = protowire.AppendTag(b, num, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

func appendMsg(b []byte, num protowire.Number, m []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, m)
}

// PairingRequest { service_name = 1; client_name = 2 }
func msgPairingRequest(clientName string) []byte {
	var p []byte
	p = appendString(p, 1, "atvremote")
	p = appendString(p, 2, clientName)
	return outer(fieldPairingRequest, p)
}

// Options.Encoding { type = 1; symbol_length = 2 }
func hexEncoding() []byte {
	var e []byte
	e = appendUint(e, 1, encodingHexadecimal)
	e = appendUint(e, 2, codeSymbols)
	return e
}

// Options { repeated Encoding input_encodings = 1; preferred_role = 3 }
func msgOptions() []byte {
	var p []byte
	p = appendMsg(p, 1, hexEncoding())
	p = appendUint(p, 3, roleInput)
	return outer(fieldOptions, p)
}

// Configuration { Encoding encoding = 1; RoleType client_role = 2 }
func msgConfiguration() []byte {
	var p []byte
	p = appendMsg(p, 1, hexEncoding())
	p = appendUint(p, 2, roleInput)
	return outer(fieldConfiguration, p)
}

// Secret { bytes secret = 1 }
func msgSecret(secret []byte) []byte {
	var p []byte
	p = protowire.AppendTag(p, 1, protowire.BytesType)
	p = protowire.AppendBytes(p, secret)
	return outer(fieldSecret, p)
}
