package test

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"net"
	"strconv"
	"testing"

	"test/nasTestpacket"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/nas/security"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/tngf/pkg/context"
	"github.com/free5gc/tngf/pkg/ike/handler"
	"github.com/free5gc/tngf/pkg/ike/message"
	"github.com/free5gc/tngf/pkg/ike/xfrm"
	radius_handler "github.com/free5gc/tngf/pkg/radius/handler"
	radius_message "github.com/free5gc/tngf/pkg/radius/message"
)

var (
	tngfInfo_IPSecIfaceAddr        = "192.168.127.1"
	tngfueInfo_IPSecIfaceAddr      = "192.168.127.2"
	tngfueInfo_SmPolicy_SNSSAI_SST = "1"
	tngfueInfo_SmPolicy_SNSSAI_SD  = "fedcba"
	tngfueInfo_IPSecIfaceName      = "veth3"
	tngfueInfo_XfrmiName           = "ipsec"
	tngfueInfo_XfrmiId             = uint32(1)
	tngfueInfo_GreIfaceName        = "gretun"
	tngfueInnerAddr                = new(net.IPNet)
)

func tngfgenerateSPI(tngfue *context.TNGFUe) []byte {
	var spi uint32
	spiByte := make([]byte, 4)
	for {
		randomUint64 := handler.GenerateRandomNumber().Uint64()
		if _, ok := tngfue.TNGFChildSecurityAssociation[uint32(randomUint64)]; !ok {
			spi = uint32(randomUint64)
			binary.BigEndian.PutUint32(spiByte, spi)
			break
		}
	}
	return spiByte
}

// func setupIPsecXfrmi(xfrmIfaceName, parentIfaceName string, xfrmIfaceId uint32, xfrmIfaceAddr *net.IPNet) (netlink.Link, error) {
// 	var (
// 		xfrmi, parent netlink.Link
// 		err           error
// 	)

// 	if parent, err = netlink.LinkByName(parentIfaceName); err != nil {
// 		return nil, err
// 	}

// 	link := &netlink.Xfrmi{
// 		LinkAttrs: netlink.LinkAttrs{
// 			MTU:         1478,
// 			Name:        xfrmIfaceName,
// 			ParentIndex: parent.Attrs().Index,
// 		},
// 		Ifid: xfrmIfaceId,
// 	}

// 	// ip link add
// 	if err := netlink.LinkAdd(link); err != nil {
// 		return nil, err
// 	}

// 	if xfrmi, err = netlink.LinkByName(xfrmIfaceName); err != nil {
// 		return nil, err
// 	}

// 	// ip addr add
// 	linkIPSecAddr := &netlink.Addr{
// 		IPNet: xfrmIfaceAddr,
// 	}

// 	if err := netlink.AddrAdd(xfrmi, linkIPSecAddr); err != nil {
// 		return nil, err
// 	}

// 	// ip link set ... up
// 	if err := netlink.LinkSetUp(xfrmi); err != nil {
// 		return nil, err
// 	}

// 	return xfrmi, nil
// }

// func setupGreTunnel(greIfaceName, parentIfaceName string, ueTunnelAddr, tngfTunnelAddr, pduAddr net.IP, qoSInfo *PDUQoSInfo, t *testing.T) (netlink.Link, error) {
// 	var (
// 		parent      netlink.Link
// 		greKeyField uint32
// 		err         error
// 	)

// 	if qoSInfo != nil {
// 		greKeyField |= (uint32(qoSInfo.qfiList[0]) & 0x3F) << 24
// 	}

// 	if parent, err = netlink.LinkByName(parentIfaceName); err != nil {
// 		return nil, err
// 	}

// 	// New GRE tunnel interface
// 	newGRETunnel := &netlink.Gretun{
// 		LinkAttrs: netlink.LinkAttrs{
// 			Name: greIfaceName,
// 			MTU:  1438, // remain for endpoint IP header(most 40 bytes if IPv6) and ESP header (22 bytes)
// 		},
// 		Link:   uint32(parent.Attrs().Index), // PHYS_DEV in iproute2; IFLA_GRE_LINK in linux kernel
// 		Local:  ueTunnelAddr,
// 		Remote: tngfTunnelAddr,
// 		IKey:   greKeyField,
// 		OKey:   greKeyField,
// 	}

// 	t.Logf("GRE Key Field: 0x%x", greKeyField)

// 	if err := netlink.LinkAdd(newGRETunnel); err != nil {
// 		return nil, err
// 	}

// 	// Get link info
// 	linkGRE, err := netlink.LinkByName(greIfaceName)
// 	if err != nil {
// 		return nil, fmt.Errorf("No link named %s", greIfaceName)
// 	}

// 	linkGREAddr := &netlink.Addr{
// 		IPNet: &net.IPNet{
// 			IP:   pduAddr,
// 			Mask: net.IPv4Mask(255, 255, 255, 255),
// 		},
// 	}

// 	if err := netlink.AddrAdd(linkGRE, linkGREAddr); err != nil {
// 		return nil, err
// 	}

// 	// Set GRE interface up
// 	if err := netlink.LinkSetUp(linkGRE); err != nil {
// 		return nil, err
// 	}

// 	return linkGRE, nil
// }

// func getAuthSubscription() (authSubs models.AuthenticationSubscription) {
// 	authSubs.PermanentKey = &models.PermanentKey{
// 		PermanentKeyValue: TestGenAuthData.MilenageTestSet19.K,
// 	}
// 	authSubs.Opc = &models.Opc{
// 		OpcValue: TestGenAuthData.MilenageTestSet19.OPC,
// 	}
// 	authSubs.Milenage = &models.Milenage{
// 		Op: &models.Op{
// 			OpValue: TestGenAuthData.MilenageTestSet19.OP,
// 		},
// 	}
// 	authSubs.AuthenticationManagementField = "8000"

// 	authSubs.SequenceNumber = TestGenAuthData.MilenageTestSet19.SQN
// 	authSubs.AuthenticationMethod = models.AuthMethod__5_G_AKA
// 	return
// }

func setupRadiusSocket() (*net.UDPConn, error) {
	bindAddr := tngfueInfo_IPSecIfaceAddr + ":48744"
	udpAddr, err := net.ResolveUDPAddr("udp", bindAddr)
	if err != nil {
		return nil, fmt.Errorf("Resolve UDP address failed: %+v", err)
	}
	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("Resolve UDP address failed: %+v", err)
	}
	return udpListener, nil
}

// func concatenateNonceAndSPI(nonce []byte, SPI_initiator uint64, SPI_responder uint64) []byte {
// 	spi := make([]byte, 8)

// 	binary.BigEndian.PutUint64(spi, SPI_initiator)
// 	newSlice := append(nonce, spi...)
// 	binary.BigEndian.PutUint64(spi, SPI_responder)
// 	newSlice = append(newSlice, spi...)

// 	return newSlice
// }

func tngfgenerateKeyForIKESA(ikeSecurityAssociation *context.IKESecurityAssociation) error {
	// Transforms
	transformPseudorandomFunction := ikeSecurityAssociation.PseudorandomFunction

	// Get key length of SK_d, SK_ai, SK_ar, SK_ei, SK_er, SK_pi, SK_pr
	var length_SK_d, length_SK_ai, length_SK_ar, length_SK_ei, length_SK_er, length_SK_pi, length_SK_pr, totalKeyLength int
	var ok bool

	length_SK_d = 20
	length_SK_ai = 20
	length_SK_ar = length_SK_ai
	length_SK_ei = 32
	length_SK_er = length_SK_ei
	length_SK_pi, length_SK_pr = length_SK_d, length_SK_d
	totalKeyLength = length_SK_d + length_SK_ai + length_SK_ar + length_SK_ei + length_SK_er + length_SK_pi + length_SK_pr

	// Generate IKE SA key as defined in RFC7296 Section 1.3 and Section 1.4
	var pseudorandomFunction hash.Hash

	if pseudorandomFunction, ok = handler.NewPseudorandomFunction(ikeSecurityAssociation.ConcatenatedNonce, transformPseudorandomFunction.TransformID); !ok {
		return errors.New("New pseudorandom function failed")
	}

	if _, err := pseudorandomFunction.Write(ikeSecurityAssociation.DiffieHellmanSharedKey); err != nil {
		return errors.New("Pseudorandom function write failed")
	}

	SKEYSEED := pseudorandomFunction.Sum(nil)

	seed := concatenateNonceAndSPI(ikeSecurityAssociation.ConcatenatedNonce, ikeSecurityAssociation.LocalSPI, ikeSecurityAssociation.RemoteSPI)

	var keyStream, generatedKeyBlock []byte
	var index byte
	for index = 1; len(keyStream) < totalKeyLength; index++ {
		if pseudorandomFunction, ok = handler.NewPseudorandomFunction(SKEYSEED, transformPseudorandomFunction.TransformID); !ok {
			return errors.New("New pseudorandom function failed")
		}
		if _, err := pseudorandomFunction.Write(append(append(generatedKeyBlock, seed...), index)); err != nil {
			return errors.New("Pseudorandom function write failed")
		}
		generatedKeyBlock = pseudorandomFunction.Sum(nil)
		keyStream = append(keyStream, generatedKeyBlock...)
	}

	// Assign keys into context
	ikeSecurityAssociation.SK_d = keyStream[:length_SK_d]
	keyStream = keyStream[length_SK_d:]
	ikeSecurityAssociation.SK_ai = keyStream[:length_SK_ai]
	keyStream = keyStream[length_SK_ai:]
	ikeSecurityAssociation.SK_ar = keyStream[:length_SK_ar]
	keyStream = keyStream[length_SK_ar:]
	ikeSecurityAssociation.SK_ei = keyStream[:length_SK_ei]
	keyStream = keyStream[length_SK_ei:]
	ikeSecurityAssociation.SK_er = keyStream[:length_SK_er]
	keyStream = keyStream[length_SK_er:]
	ikeSecurityAssociation.SK_pi = keyStream[:length_SK_pi]
	keyStream = keyStream[length_SK_pi:]
	ikeSecurityAssociation.SK_pr = keyStream[:length_SK_pr]
	keyStream = keyStream[length_SK_pr:]

	return nil
}

func tngfgenerateKeyForChildSA(ikeSecurityAssociation *context.IKESecurityAssociation, childSecurityAssociation *context.ChildSecurityAssociation) error {
	// Transforms
	transformPseudorandomFunction := ikeSecurityAssociation.PseudorandomFunction
	var transformIntegrityAlgorithmForIPSec *message.Transform
	if len(ikeSecurityAssociation.IKEAuthResponseSA.Proposals[0].IntegrityAlgorithm) != 0 {
		transformIntegrityAlgorithmForIPSec = ikeSecurityAssociation.IKEAuthResponseSA.Proposals[0].IntegrityAlgorithm[0]
	}

	// Get key length for encryption and integrity key for IPSec
	var lengthEncryptionKeyIPSec, lengthIntegrityKeyIPSec, totalKeyLength int
	var ok bool

	lengthEncryptionKeyIPSec = 32
	if transformIntegrityAlgorithmForIPSec != nil {
		lengthIntegrityKeyIPSec = 20
	}
	totalKeyLength = lengthEncryptionKeyIPSec + lengthIntegrityKeyIPSec
	totalKeyLength = totalKeyLength * 2

	// Generate key for child security association as specified in RFC 7296 section 2.17
	seed := ikeSecurityAssociation.ConcatenatedNonce
	var pseudorandomFunction hash.Hash

	var keyStream, generatedKeyBlock []byte
	var index byte
	for index = 1; len(keyStream) < totalKeyLength; index++ {
		if pseudorandomFunction, ok = handler.NewPseudorandomFunction(ikeSecurityAssociation.SK_d, transformPseudorandomFunction.TransformID); !ok {
			return errors.New("New pseudorandom function failed")
		}
		if _, err := pseudorandomFunction.Write(append(append(generatedKeyBlock, seed...), index)); err != nil {
			return errors.New("Pseudorandom function write failed")
		}
		generatedKeyBlock = pseudorandomFunction.Sum(nil)
		keyStream = append(keyStream, generatedKeyBlock...)
	}

	childSecurityAssociation.InitiatorToResponderEncryptionKey = append(childSecurityAssociation.InitiatorToResponderEncryptionKey, keyStream[:lengthEncryptionKeyIPSec]...)
	keyStream = keyStream[lengthEncryptionKeyIPSec:]
	childSecurityAssociation.InitiatorToResponderIntegrityKey = append(childSecurityAssociation.InitiatorToResponderIntegrityKey, keyStream[:lengthIntegrityKeyIPSec]...)
	keyStream = keyStream[lengthIntegrityKeyIPSec:]
	childSecurityAssociation.ResponderToInitiatorEncryptionKey = append(childSecurityAssociation.ResponderToInitiatorEncryptionKey, keyStream[:lengthEncryptionKeyIPSec]...)
	keyStream = keyStream[lengthEncryptionKeyIPSec:]
	childSecurityAssociation.ResponderToInitiatorIntegrityKey = append(childSecurityAssociation.ResponderToInitiatorIntegrityKey, keyStream[:lengthIntegrityKeyIPSec]...)

	return nil

}

func tngfdecryptProcedure(ikeSecurityAssociation *context.IKESecurityAssociation, ikeMessage *message.IKEMessage, encryptedPayload *message.Encrypted) (message.IKEPayloadContainer, error) {
	// Load needed information
	transformIntegrityAlgorithm := ikeSecurityAssociation.IntegrityAlgorithm
	transformEncryptionAlgorithm := ikeSecurityAssociation.EncryptionAlgorithm
	checksumLength := 12 // HMAC_SHA1_96

	// Checksum
	checksum := encryptedPayload.EncryptedData[len(encryptedPayload.EncryptedData)-checksumLength:]

	ikeMessageData, err := ikeMessage.Encode()
	if err != nil {
		return nil, errors.New("Encoding IKE message failed")
	}

	ok, err := handler.VerifyIKEChecksum(ikeSecurityAssociation.SK_ar, ikeMessageData[:len(ikeMessageData)-checksumLength], checksum, transformIntegrityAlgorithm.TransformID)
	if err != nil {
		return nil, errors.New("Error verify checksum")
	}
	if !ok {
		return nil, errors.New("Checksum failed, drop.")
	}

	// Decrypt
	encryptedData := encryptedPayload.EncryptedData[:len(encryptedPayload.EncryptedData)-checksumLength]
	plainText, err := handler.DecryptMessage(ikeSecurityAssociation.SK_er, encryptedData, transformEncryptionAlgorithm.TransformID)
	if err != nil {
		return nil, errors.New("Error decrypting message")
	}

	var decryptedIKEPayload message.IKEPayloadContainer
	err = decryptedIKEPayload.Decode(encryptedPayload.NextPayload, plainText)
	if err != nil {
		return nil, errors.New("Decoding decrypted payload failed")
	}

	return decryptedIKEPayload, nil

}

func tngfencryptProcedure(ikeSecurityAssociation *context.IKESecurityAssociation, ikePayload message.IKEPayloadContainer, responseIKEMessage *message.IKEMessage) error {
	// Load needed information
	transformIntegrityAlgorithm := ikeSecurityAssociation.IntegrityAlgorithm
	transformEncryptionAlgorithm := ikeSecurityAssociation.EncryptionAlgorithm
	checksumLength := 12 // HMAC_SHA1_96

	// Encrypting
	notificationPayloadData, err := ikePayload.Encode()
	if err != nil {
		return errors.New("Encoding IKE payload failed.")
	}

	encryptedData, err := handler.EncryptMessage(ikeSecurityAssociation.SK_ei, notificationPayloadData, transformEncryptionAlgorithm.TransformID)
	if err != nil {
		return errors.New("Error encrypting message")
	}

	encryptedData = append(encryptedData, make([]byte, checksumLength)...)
	sk := responseIKEMessage.Payloads.BuildEncrypted(ikePayload[0].Type(), encryptedData)

	// Calculate checksum
	responseIKEMessageData, err := responseIKEMessage.Encode()
	if err != nil {
		return errors.New("Encoding IKE message error")
	}
	checksumOfMessage, err := handler.CalculateChecksum(ikeSecurityAssociation.SK_ai, responseIKEMessageData[:len(responseIKEMessageData)-checksumLength], transformIntegrityAlgorithm.TransformID)
	if err != nil {
		return errors.New("Error calculating checksum")
	}
	checksumField := sk.EncryptedData[len(sk.EncryptedData)-checksumLength:]
	copy(checksumField, checksumOfMessage)

	return nil

}

// [TS 24502] 9.3.2.2.2 EAP-Response/5G-NAS message
// Define EAP-Response/5G-NAS message and AN-Parameters Format.

// [TS 24501] 8.2.6.1.1  REGISTRATION REQUEST message content
// For dealing with EAP-5G start, return EAP-5G response including
// "AN-Parameters and NASPDU of Registration Request"

// func buildEAP5GANParameters() []byte {
// 	var anParameters []byte

// 	// [TS 24.502] 9.3.2.2.2.3
// 	// AN-parameter value field in GUAMI, PLMN ID and NSSAI is coded as value part
// 	// Therefore, IEI of AN-parameter is not needed to be included.

// 	// anParameter = AN-parameter Type | AN-parameter Length | Value part of IE

// 	// Build GUAMI
// 	anParameter := make([]byte, 2)
// 	guami := make([]byte, 6)
// 	guami[0] = 0x02
// 	guami[1] = 0xf8
// 	guami[2] = 0x39
// 	guami[3] = 0xca
// 	guami[4] = 0xfe
// 	guami[5] = 0x0
// 	anParameter[0] = message.ANParametersTypeGUAMI
// 	anParameter[1] = byte(len(guami))
// 	anParameter = append(anParameter, guami...)

// 	anParameters = append(anParameters, anParameter...)

// 	// Build Establishment Cause
// 	anParameter = make([]byte, 2)
// 	establishmentCause := make([]byte, 1)
// 	establishmentCause[0] = message.EstablishmentCauseMO_Signalling
// 	anParameter[0] = message.ANParametersTypeEstablishmentCause
// 	anParameter[1] = byte(len(establishmentCause))
// 	anParameter = append(anParameter, establishmentCause...)

// 	anParameters = append(anParameters, anParameter...)

// 	// Build PLMN ID
// 	anParameter = make([]byte, 2)
// 	plmnID := make([]byte, 3)
// 	plmnID[0] = 0x02
// 	plmnID[1] = 0xf8
// 	plmnID[2] = 0x39
// 	anParameter[0] = message.ANParametersTypeSelectedPLMNID
// 	anParameter[1] = byte(len(plmnID))
// 	anParameter = append(anParameter, plmnID...)

// 	anParameters = append(anParameters, anParameter...)

// 	// Build NSSAI
// 	anParameter = make([]byte, 2)
// 	var nssai []byte
// 	// s-nssai = s-nssai length(1 byte) | SST(1 byte) | SD(3 bytes)
// 	snssai := make([]byte, 5)
// 	snssai[0] = 4
// 	snssai[1] = 1
// 	snssai[2] = 0x01
// 	snssai[3] = 0x02
// 	snssai[4] = 0x03
// 	nssai = append(nssai, snssai...)
// 	snssai = make([]byte, 5)
// 	snssai[0] = 4
// 	snssai[1] = 1
// 	snssai[2] = 0x11
// 	snssai[3] = 0x22
// 	snssai[4] = 0x33
// 	nssai = append(nssai, snssai...)
// 	anParameter[0] = message.ANParametersTypeRequestedNSSAI
// 	anParameter[1] = byte(len(nssai))
// 	anParameter = append(anParameter, nssai...)

// 	anParameters = append(anParameters, anParameter...)

// 	return anParameters
// }

func tngfparseIPAddressInformationToChildSecurityAssociation(
	childSecurityAssociation *context.ChildSecurityAssociation,
	trafficSelectorLocal *message.IndividualTrafficSelector,
	trafficSelectorRemote *message.IndividualTrafficSelector) error {

	if childSecurityAssociation == nil {
		return errors.New("childSecurityAssociation is nil")
	}

	childSecurityAssociation.PeerPublicIPAddr = net.ParseIP(tngfInfo_IPSecIfaceAddr)
	childSecurityAssociation.LocalPublicIPAddr = net.ParseIP(tngfueInfo_IPSecIfaceAddr)

	childSecurityAssociation.TrafficSelectorLocal = net.IPNet{
		IP:   trafficSelectorLocal.StartAddress,
		Mask: []byte{255, 255, 255, 255},
	}

	childSecurityAssociation.TrafficSelectorRemote = net.IPNet{
		IP:   trafficSelectorRemote.StartAddress,
		Mask: []byte{255, 255, 255, 255},
	}

	return nil
}

// type PDUQoSInfo struct {
// 	pduSessionID    uint8
// 	qfiList         []uint8
// 	isDefault       bool
// 	isDSCPSpecified bool
// 	DSCP            uint8
// }

func tngfparse5GQoSInfoNotify(n *message.Notification) (info *PDUQoSInfo, err error) {
	info = new(PDUQoSInfo)
	var offset int = 0
	data := n.NotificationData
	dataLen := int(data[0])
	info.pduSessionID = data[1]
	qfiListLen := int(data[2])
	offset += (3 + qfiListLen)

	if offset > dataLen {
		return nil, errors.New("parse5GQoSInfoNotify err: Length and content of 5G-QoS-Info-Notify mismatch")
	}

	info.qfiList = make([]byte, qfiListLen)
	copy(info.qfiList, data[3:3+qfiListLen])

	info.isDefault = (data[offset] & message.NotifyType5G_QOS_INFOBitDCSICheck) > 0
	info.isDSCPSpecified = (data[offset] & message.NotifyType5G_QOS_INFOBitDSCPICheck) > 0

	return
}

func tngfapplyXFRMRule(ue_is_initiator bool, ifId uint32, childSecurityAssociation *context.ChildSecurityAssociation) error {
	// Build XFRM information data structure for incoming traffic.

	// Mark
	// mark := &netlink.XfrmMark{
	// 	Value: ifMark, // tngfueInfo.XfrmMark,
	// }

	// Direction: TNGF -> UE
	// State
	var xfrmEncryptionAlgorithm, xfrmIntegrityAlgorithm *netlink.XfrmStateAlgo
	if ue_is_initiator {
		xfrmEncryptionAlgorithm = &netlink.XfrmStateAlgo{
			Name: xfrm.XFRMEncryptionAlgorithmType(childSecurityAssociation.EncryptionAlgorithm).String(),
			Key:  childSecurityAssociation.ResponderToInitiatorEncryptionKey,
		}
		if childSecurityAssociation.IntegrityAlgorithm != 0 {
			xfrmIntegrityAlgorithm = &netlink.XfrmStateAlgo{
				Name: xfrm.XFRMIntegrityAlgorithmType(childSecurityAssociation.IntegrityAlgorithm).String(),
				Key:  childSecurityAssociation.ResponderToInitiatorIntegrityKey,
			}
		}
	} else {
		xfrmEncryptionAlgorithm = &netlink.XfrmStateAlgo{
			Name: xfrm.XFRMEncryptionAlgorithmType(childSecurityAssociation.EncryptionAlgorithm).String(),
			Key:  childSecurityAssociation.InitiatorToResponderEncryptionKey,
		}
		if childSecurityAssociation.IntegrityAlgorithm != 0 {
			xfrmIntegrityAlgorithm = &netlink.XfrmStateAlgo{
				Name: xfrm.XFRMIntegrityAlgorithmType(childSecurityAssociation.IntegrityAlgorithm).String(),
				Key:  childSecurityAssociation.InitiatorToResponderIntegrityKey,
			}
		}
	}

	xfrmState := new(netlink.XfrmState)

	xfrmState.Src = childSecurityAssociation.PeerPublicIPAddr
	xfrmState.Dst = childSecurityAssociation.LocalPublicIPAddr
	xfrmState.Proto = netlink.XFRM_PROTO_ESP
	xfrmState.Mode = netlink.XFRM_MODE_TUNNEL
	xfrmState.Spi = int(childSecurityAssociation.InboundSPI)
	xfrmState.Ifid = int(ifId)
	xfrmState.Auth = xfrmIntegrityAlgorithm
	xfrmState.Crypt = xfrmEncryptionAlgorithm
	xfrmState.ESN = childSecurityAssociation.ESN

	// Commit xfrm state to netlink
	var err error
	if err = netlink.XfrmStateAdd(xfrmState); err != nil {
		return fmt.Errorf("Set XFRM state rule failed: %+v", err)
	}

	// Policy
	xfrmPolicyTemplate := netlink.XfrmPolicyTmpl{
		Src:   xfrmState.Src,
		Dst:   xfrmState.Dst,
		Proto: xfrmState.Proto,
		Mode:  xfrmState.Mode,
		Spi:   xfrmState.Spi,
	}

	xfrmPolicy := new(netlink.XfrmPolicy)

	if childSecurityAssociation.SelectedIPProtocol == 0 {
		return errors.New("Protocol == 0")
	}

	xfrmPolicy.Src = &childSecurityAssociation.TrafficSelectorRemote
	xfrmPolicy.Dst = &childSecurityAssociation.TrafficSelectorLocal
	xfrmPolicy.Proto = netlink.Proto(childSecurityAssociation.SelectedIPProtocol)
	xfrmPolicy.Dir = netlink.XFRM_DIR_IN
	xfrmPolicy.Ifid = int(ifId)
	xfrmPolicy.Tmpls = []netlink.XfrmPolicyTmpl{
		xfrmPolicyTemplate,
	}

	// Commit xfrm policy to netlink
	if err = netlink.XfrmPolicyAdd(xfrmPolicy); err != nil {
		return fmt.Errorf("Set XFRM policy rule failed: %+v", err)
	}

	// Direction: UE -> TNGF
	// State
	if ue_is_initiator {
		xfrmEncryptionAlgorithm.Key = childSecurityAssociation.InitiatorToResponderEncryptionKey
		if childSecurityAssociation.IntegrityAlgorithm != 0 {
			xfrmIntegrityAlgorithm.Key = childSecurityAssociation.InitiatorToResponderIntegrityKey
		}
	} else {
		xfrmEncryptionAlgorithm.Key = childSecurityAssociation.ResponderToInitiatorEncryptionKey
		if childSecurityAssociation.IntegrityAlgorithm != 0 {
			xfrmIntegrityAlgorithm.Key = childSecurityAssociation.ResponderToInitiatorIntegrityKey
		}
	}

	xfrmState.Src, xfrmState.Dst = xfrmState.Dst, xfrmState.Src
	xfrmState.Spi = int(childSecurityAssociation.OutboundSPI)

	// Commit xfrm state to netlink
	if err = netlink.XfrmStateAdd(xfrmState); err != nil {
		return fmt.Errorf("Set XFRM state rule failed: %+v", err)
	}

	// Policy
	xfrmPolicyTemplate.Src, xfrmPolicyTemplate.Dst = xfrmPolicyTemplate.Dst, xfrmPolicyTemplate.Src
	xfrmPolicyTemplate.Spi = int(childSecurityAssociation.OutboundSPI)

	xfrmPolicy.Src, xfrmPolicy.Dst = xfrmPolicy.Dst, xfrmPolicy.Src
	xfrmPolicy.Dir = netlink.XFRM_DIR_OUT
	xfrmPolicy.Tmpls = []netlink.XfrmPolicyTmpl{
		xfrmPolicyTemplate,
	}

	// Commit xfrm policy to netlink
	if err = netlink.XfrmPolicyAdd(xfrmPolicy); err != nil {
		return fmt.Errorf("Set XFRM policy rule failed: %+v", err)
	}

	return nil
}

func tngfsendPduSessionEstablishmentRequest(
	pduSessionId uint8,
	ue *RanUeContext,
	n3Info *context.TNGFUe,
	ikeSA *context.IKESecurityAssociation,
	ikeConn *net.UDPConn,
	nasConn *net.TCPConn,
	t *testing.T) ([]netlink.Link, error) {

	var ifaces []netlink.Link

	// Build S-NSSA
	sst, err := strconv.ParseInt(tngfueInfo_SmPolicy_SNSSAI_SST, 16, 0)

	if err != nil {
		return ifaces, fmt.Errorf("Parse SST Fail:%+v", err)
	}

	sNssai := models.Snssai{
		Sst: int32(sst),
		Sd:  tngfueInfo_SmPolicy_SNSSAI_SD,
	}

	// PDU session establishment request
	// TS 24.501 9.11.3.47.1 Request type
	pdu := nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(pduSessionId, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)
	pdu, err = EncodeNasPduInEnvelopeWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		return ifaces, fmt.Errorf("Encode NAS PDU In Envelope Fail:%+v", err)
	}
	if _, err = nasConn.Write(pdu); err != nil {
		return ifaces, fmt.Errorf("Send NAS Message Fail:%+v", err)
	}

	buffer := make([]byte, 65535)

	t.Logf("Waiting for TNGF reply from IKE")

	// Receive TNGF reply
	n, _, err := ikeConn.ReadFromUDP(buffer)
	if err != nil {
		return ifaces, fmt.Errorf("Read IKE Message Fail:%+v", err)
	}

	ikeMessage := new(message.IKEMessage)
	ikeMessage.Payloads.Reset()
	err = ikeMessage.Decode(buffer[:n])
	if err != nil {
		return ifaces, fmt.Errorf("Decode IKE Message Fail:%+v", err)
	}
	t.Logf("IKE message exchange type: %d", ikeMessage.ExchangeType)
	t.Logf("IKE message ID: %d", ikeMessage.MessageID)

	encryptedPayload, ok := ikeMessage.Payloads[0].(*message.Encrypted)
	if !ok {
		return ifaces, errors.New("Received pakcet is not an encrypted payload")
	}
	decryptedIKEPayload, err := tngfdecryptProcedure(ikeSA, ikeMessage, encryptedPayload)
	if err != nil {
		return ifaces, fmt.Errorf("Decrypt IKE Message Fail:%+v", err)
	}

	var qoSInfo *PDUQoSInfo

	var responseSecurityAssociation *message.SecurityAssociation
	var responseTrafficSelectorInitiator *message.TrafficSelectorInitiator
	var responseTrafficSelectorResponder *message.TrafficSelectorResponder
	var outboundSPI uint32
	var upIPAddr net.IP
	for _, ikePayload := range decryptedIKEPayload {
		switch ikePayload.Type() {
		case message.TypeSA:
			responseSecurityAssociation = ikePayload.(*message.SecurityAssociation)
			outboundSPI = binary.BigEndian.Uint32(responseSecurityAssociation.Proposals[0].SPI)
		case message.TypeTSi:
			responseTrafficSelectorInitiator = ikePayload.(*message.TrafficSelectorInitiator)
		case message.TypeTSr:
			responseTrafficSelectorResponder = ikePayload.(*message.TrafficSelectorResponder)
		case message.TypeN:
			notification := ikePayload.(*message.Notification)
			if notification.NotifyMessageType == message.Vendor3GPPNotifyType5G_QOS_INFO {
				t.Logf("Received Qos Flow settings")
				if info, err := tngfparse5GQoSInfoNotify(notification); err == nil {
					qoSInfo = info
					t.Logf("NotificationData:%+v", notification.NotificationData)
					if qoSInfo.isDSCPSpecified {
						t.Logf("DSCP is specified but test not support")
					}
				} else {
					t.Logf("%+v", err)
				}
			}
			if notification.NotifyMessageType == message.Vendor3GPPNotifyTypeUP_IP4_ADDRESS {
				upIPAddr = notification.NotificationData[:4]
				t.Logf("UP IP Address: %+v\n", upIPAddr)
			}
		case message.TypeNiNr:
			responseNonce := ikePayload.(*message.Nonce)
			ikeSA.ConcatenatedNonce = responseNonce.NonceData
		}
	}

	// IKE CREATE_CHILD_SA response
	ikeMessage.Payloads.Reset()
	n3Info.TNGFIKESecurityAssociation.ResponderMessageID = ikeMessage.MessageID
	ikeMessage.BuildIKEHeader(ikeMessage.InitiatorSPI, ikeMessage.ResponderSPI,
		message.CREATE_CHILD_SA, message.ResponseBitCheck|message.InitiatorBitCheck,
		n3Info.TNGFIKESecurityAssociation.ResponderMessageID)

	var ikePayload message.IKEPayloadContainer
	ikePayload.Reset()

	// SA
	inboundSPI := tngfgenerateSPI(n3Info)
	responseSecurityAssociation.Proposals[0].SPI = inboundSPI
	ikePayload = append(ikePayload, responseSecurityAssociation)

	// TSi
	ikePayload = append(ikePayload, responseTrafficSelectorInitiator)

	// TSr
	ikePayload = append(ikePayload, responseTrafficSelectorResponder)

	// Nonce
	localNonce := handler.GenerateRandomNumber().Bytes()
	ikeSA.ConcatenatedNonce = append(ikeSA.ConcatenatedNonce, localNonce...)
	ikePayload.BuildNonce(localNonce)

	if err := tngfencryptProcedure(ikeSA, ikePayload, ikeMessage); err != nil {
		t.Errorf("Encrypt IKE message failed: %+v", err)
		return ifaces, err
	}

	// Send to TNGF
	ikeMessageData, err := ikeMessage.Encode()
	if err != nil {
		return ifaces, fmt.Errorf("Encode IKE Message Fail:%+v", err)
	}

	tngfUDPAddr, err := net.ResolveUDPAddr("udp", tngfInfo_IPSecIfaceAddr+":500")

	if err != nil {
		return ifaces, fmt.Errorf("Resolve TNGF IPSec IP Addr Fail:%+v", err)
	}

	_, err = ikeConn.WriteToUDP(ikeMessageData, tngfUDPAddr)
	if err != nil {
		t.Errorf("Write IKE maessage fail: %+v", err)
		return ifaces, err
	}

	n3Info.CreateHalfChildSA(n3Info.TNGFIKESecurityAssociation.ResponderMessageID, binary.BigEndian.Uint32(inboundSPI), int64(pduSessionId))
	childSecurityAssociationContextUserPlane, err := n3Info.CompleteChildSA(
		n3Info.TNGFIKESecurityAssociation.ResponderMessageID, outboundSPI, responseSecurityAssociation)
	if err != nil {
		return ifaces, fmt.Errorf("Create child security association context failed: %+v", err)
	}

	err = tngfparseIPAddressInformationToChildSecurityAssociation(
		childSecurityAssociationContextUserPlane,
		responseTrafficSelectorResponder.TrafficSelectors[0],
		responseTrafficSelectorInitiator.TrafficSelectors[0])

	if err != nil {
		return ifaces, fmt.Errorf("Parse IP address to child security association failed: %+v", err)
	}
	// Select GRE traffic
	childSecurityAssociationContextUserPlane.SelectedIPProtocol = unix.IPPROTO_GRE

	if err := tngfgenerateKeyForChildSA(ikeSA, childSecurityAssociationContextUserPlane); err != nil {
		return ifaces, fmt.Errorf("Generate key for child SA failed: %+v", err)
	}

	// ====== Inbound ======
	t.Logf("====== IPSec/Child SA for 3GPP UP Inbound =====")
	t.Logf("[UE:%+v] <- [TNGF:%+v]",
		childSecurityAssociationContextUserPlane.LocalPublicIPAddr, childSecurityAssociationContextUserPlane.PeerPublicIPAddr)
	t.Logf("IPSec SPI: 0x%016x", childSecurityAssociationContextUserPlane.InboundSPI)
	t.Logf("IPSec Encryption Algorithm: %d", childSecurityAssociationContextUserPlane.EncryptionAlgorithm)
	t.Logf("IPSec Encryption Key: 0x%x", childSecurityAssociationContextUserPlane.InitiatorToResponderEncryptionKey)
	t.Logf("IPSec Integrity  Algorithm: %d", childSecurityAssociationContextUserPlane.IntegrityAlgorithm)
	t.Logf("IPSec Integrity  Key: 0x%x", childSecurityAssociationContextUserPlane.InitiatorToResponderIntegrityKey)
	// ====== Outbound ======
	t.Logf("====== IPSec/Child SA for 3GPP UP Outbound =====")
	t.Logf("[UE:%+v] -> [TNGF:%+v]",
		childSecurityAssociationContextUserPlane.LocalPublicIPAddr, childSecurityAssociationContextUserPlane.PeerPublicIPAddr)
	t.Logf("IPSec SPI: 0x%016x", childSecurityAssociationContextUserPlane.OutboundSPI)
	t.Logf("IPSec Encryption Algorithm: %d", childSecurityAssociationContextUserPlane.EncryptionAlgorithm)
	t.Logf("IPSec Encryption Key: 0x%x", childSecurityAssociationContextUserPlane.ResponderToInitiatorEncryptionKey)
	t.Logf("IPSec Integrity  Algorithm: %d", childSecurityAssociationContextUserPlane.IntegrityAlgorithm)
	t.Logf("IPSec Integrity  Key: 0x%x", childSecurityAssociationContextUserPlane.ResponderToInitiatorIntegrityKey)
	t.Logf("State function: encr: %d, auth: %d", childSecurityAssociationContextUserPlane.EncryptionAlgorithm, childSecurityAssociationContextUserPlane.IntegrityAlgorithm)

	// Aplly XFRM rules
	tngfueInfo_XfrmiId++
	err = tngfapplyXFRMRule(false, tngfueInfo_XfrmiId, childSecurityAssociationContextUserPlane)

	if err != nil {
		t.Errorf("Applying XFRM rules failed: %+v", err)
		return ifaces, err
	}

	var linkIPSec netlink.Link

	// Setup interface for ipsec
	newXfrmiName := fmt.Sprintf("%s-%d", tngfueInfo_XfrmiName, tngfueInfo_XfrmiId)
	if linkIPSec, err = setupIPsecXfrmi(newXfrmiName, tngfueInfo_IPSecIfaceName, tngfueInfo_XfrmiId, tngfueInnerAddr); err != nil {
		return ifaces, fmt.Errorf("Setup XFRMi interface %s fail: %+v", newXfrmiName, err)
	}

	ifaces = append(ifaces, linkIPSec)

	t.Logf("Setup XFRM interface %s successfully", newXfrmiName)

	var pduAddr net.IP

	// Read NAS from TNGF
	if n, err := nasConn.Read(buffer); err != nil {
		return ifaces, fmt.Errorf("Read NAS Message Fail:%+v", err)
	} else {
		nasMsg, err := DecodePDUSessionEstablishmentAccept(ue, n, buffer)
		if err != nil {
			t.Errorf("DecodePDUSessionEstablishmentAccept Fail: %+v", err)
		}
		spew.Config.Indent = "\t"
		nasStr := spew.Sdump(nasMsg)
		t.Log("Dump DecodePDUSessionEstablishmentAccept:\n", nasStr)

		pduAddr, err = GetPDUAddress(nasMsg.GsmMessage.PDUSessionEstablishmentAccept)
		if err != nil {
			t.Errorf("GetPDUAddress Fail: %+v", err)
		}

		t.Logf("PDU Address: %s", pduAddr.String())
	}

	var linkGRE netlink.Link

	newGREName := fmt.Sprintf("%s-id-%d", tngfueInfo_GreIfaceName, tngfueInfo_XfrmiId)

	if linkGRE, err = setupGreTunnel(newGREName, newXfrmiName, tngfueInnerAddr.IP, upIPAddr, pduAddr, qoSInfo, t); err != nil {
		return ifaces, fmt.Errorf("Setup GRE tunnel %s Fail %+v", newGREName, err)
	}

	ifaces = append(ifaces, linkGRE)

	return ifaces, nil
}

// create EAP Identity and append to Radius payload
func BuildEAPIdentity(container *radius_message.RadiusPayloadContainer, identifier uint8, identityData []byte) {
	eap := new(radius_message.EAP)
	eap.Code = radius_message.EAPCodeResponse
	eap.Identifier = identifier
	eapIdentity := new(radius_message.EAPIdentity)
	eapIdentity.IdentityData = identityData
	eap.EAPTypeData = append(eap.EAPTypeData, eapIdentity)
	eapPayload, err := eap.Marshal()
	if err != nil {
		return
	}
	payload := new(radius_message.RadiusPayload)
	payload.Type = radius_message.TypeEAPMessage
	payload.Val = eapPayload

	*container = append(*container, *payload)
}

func BuildEAP5GNAS(container *radius_message.RadiusPayloadContainer, identifier uint8, vendorData []byte) {
	eap := new(radius_message.EAP)
	eap.Code = radius_message.EAPCodeResponse
	eap.Identifier = identifier
	eap.EAPTypeData.BuildEAPExpanded(radius_message.VendorID3GPP, radius_message.VendorTypeEAP5G, vendorData)
	eapPayload, err := eap.Marshal()
	if err != nil {
		return
	}

	payload := new(radius_message.RadiusPayload)
	payload.Type = radius_message.TypeEAPMessage
	payload.Val = eapPayload

	*container = append(*container, *payload)
}

func BuildEAP5GNotification(container *radius_message.RadiusPayloadContainer, identifier uint8) {
	eap := new(radius_message.EAP)
	eap.Code = radius_message.EAPCodeResponse
	eap.Identifier = identifier
	vendorData := make([]byte, 2)
	vendorData[0] = radius_message.EAP5GType5GNotification
	eap.EAPTypeData.BuildEAPExpanded(radius_message.VendorID3GPP, radius_message.VendorTypeEAP5G, vendorData)
	eapPayload, err := eap.Marshal()
	if err != nil {
		return
	}

	payload := new(radius_message.RadiusPayload)
	payload.Type = radius_message.TypeEAPMessage
	payload.Val = eapPayload

	*container = append(*container, *payload)
}

func UEencode(radiusMessage *radius_message.RadiusMessage) ([]byte, error) {

	radiusMessageData := make([]byte, 4)

	radiusMessageData[0] = radiusMessage.Code
	radiusMessageData[1] = radiusMessage.PktID
	radiusMessageData = append(radiusMessageData, radiusMessage.Auth...)

	radiusMessagePayloadData, err := radiusMessage.Payloads.Encode()
	if err != nil {
		return nil, fmt.Errorf("Encode(): EncodePayload failed: %+v", err)
	}
	radiusMessageData = append(radiusMessageData, radiusMessagePayloadData...)
	binary.BigEndian.PutUint16(radiusMessageData[2:4], uint16(len(radiusMessageData)))

	return radiusMessageData, nil
}

func GetMessageAuthenticator(message *radius_message.RadiusMessage) []byte {
	radius_secret := []byte("free5gctngf")
	radiusMessageData := make([]byte, 4)

	radiusMessageData[0] = message.Code
	radiusMessageData[1] = message.PktID
	radiusMessageData = append(radiusMessageData, message.Auth...)

	radiusMessagePayloadData, err := message.Payloads.Encode()
	if err != nil {
		return nil
	}
	radiusMessageData = append(radiusMessageData, radiusMessagePayloadData...)
	binary.BigEndian.PutUint16(radiusMessageData[2:4], uint16(len(radiusMessageData)))

	hmacFun := hmac.New(md5.New, radius_secret) // radius_secret is same as cfg's radius_secret
	hmacFun.Write(radiusMessageData)
	return hmacFun.Sum(nil)
}

func TestTngfUE(t *testing.T) {
	// New UE
	ue := NewRanUeContext("imsi-2089300007487", 1, security.AlgCiphering128NEA0, security.AlgIntegrity128NIA2,
		models.AccessType_NON_3_GPP_ACCESS)
	ue.AmfUeNgapId = 1
	ue.AuthenticationSubs = getAuthSubscription()
	mobileIdentity5GS := nasType.MobileIdentity5GS{
		Len:    12, // suci
		Buffer: []uint8{0x01, 0x02, 0xf8, 0x39, 0xf0, 0xff, 0x00, 0x00, 0x00, 0x00, 0x47, 0x78},
	}

	// Used to save IPsec/IKE related data
	tngfue := context.TNGFSelf().NewTngfUe()
	tngfue.PduSessionList = make(map[int64]*context.PDUSession)
	tngfue.TNGFChildSecurityAssociation = make(map[uint32]*context.ChildSecurityAssociation)
	tngfue.TemporaryExchangeMsgIDChildSAMapping = make(map[uint32]*context.ChildSecurityAssociation)

	tngfRadiusUDPAddr, err := net.ResolveUDPAddr("udp", tngfInfo_IPSecIfaceAddr+":1812")
	if err != nil {
		t.Fatalf("Resolve UDP address %s fail: %+v", tngfInfo_IPSecIfaceAddr+":1812", err)
	}
	// tngfUDPAddr, err := net.ResolveUDPAddr("udp", tngfInfo_IPSecIfaceAddr+":500")
	if err != nil {
		t.Fatalf("Resolve UDP address %s fail: %+v", tngfInfo_IPSecIfaceAddr+":500", err)
	}
	// ueUDPAddr, err := net.ResolveUDPAddr("udp", tngfueInfo_IPSecIfaceAddr+":48744")
	// if err != nil {
	// 	t.Fatalf("Resolve UDP address %s fail: %+v", tngfueInfo_IPSecIfaceAddr+":48744", err)
	// }
	// udpConnection, err := setupUDPSocket()
	radiusConnection, err := setupRadiusSocket()

	if err != nil {
		t.Fatalf("Setup UDP socket Fail: %+v", err)
	}

	// calling station payload
	callingStationPayload := new(radius_message.RadiusPayload)
	callingStationPayload.Type = radius_message.TypeCallingStationId
	callingStationPayload.Length = uint8(19)
	callingStationPayload.Val = []byte("C4-85-08-77-A7-D1")
	// called station payload
	calledStationPayload := new(radius_message.RadiusPayload)
	calledStationPayload.Type = radius_message.TypeCalledStationId
	calledStationPayload.Length = uint8(30)
	calledStationPayload.Val = []byte("D4-6E-0E-65-AC-A2:free5gc-ap")
	// UE user name payload
	ueUserNamePayload := new(radius_message.RadiusPayload)
	ueUserNamePayload.Type = radius_message.TypeUserName
	ueUserNamePayload.Length = uint8(8)
	ueUserNamePayload.Val = []byte("tngfue")

	var pkt []byte

	// Step3: AAA message, send to tngf
	// create a new radius message
	ueRadiusMessage := new(radius_message.RadiusMessage)
	radiusAuthenticator := make([]byte, 16)
	rand.Read(radiusAuthenticator) // request authenticator is random
	if err != nil {
		fmt.Printf("Failed to decode hex string: %v\n", err)
		return
	}

	ueRadiusMessage.BuildRadiusHeader(radius_message.AccessRequest, 0x05, radiusAuthenticator)
	// create Radius payload
	ueRadiusPayload := new(radius_message.RadiusPayloadContainer)
	*ueRadiusPayload = append(*ueRadiusPayload, *ueUserNamePayload, *calledStationPayload, *callingStationPayload)

	// create EAP message (Identity) payload
	identifier, err := radius_handler.GenerateRandomUint8()
	if err != nil {
		t.Errorf("Random number failed: %+v", err)
		return
	}
	BuildEAPIdentity(ueRadiusPayload, identifier, []byte("tngfue"))

	// create Authenticator payload
	authPayload := new(radius_message.RadiusPayload)
	authPayload.Type = radius_message.TypeMessageAuthenticator
	authPayload.Length = uint8(18)
	authPayload.Val = make([]byte, 16)

	ueRadiusMessage.Payloads = *ueRadiusPayload
	ueRadiusMessage.Payloads = append(ueRadiusMessage.Payloads, *authPayload)
	authPayload.Val = GetMessageAuthenticator(ueRadiusMessage)
	*ueRadiusPayload = append(*ueRadiusPayload, *authPayload)
	ueRadiusMessage.Payloads = *ueRadiusPayload

	pkt, err = UEencode(ueRadiusMessage)

	if err != nil {
		t.Fatalf("Radius Message Encoding error: %+v", err)
	}
	// send to tngf
	if _, err := radiusConnection.WriteToUDP(pkt, tngfRadiusUDPAddr); err != nil {
		t.Fatalf("Write Radius maessage fail: %+v", err)
	}
	// radius_handler.SendRadiusMessageToUE(radiusConnection, ueUDPAddr, tngfRadiusUDPAddr, ueRadiusMessage)

	// Step 4: receive TNGF reply
	buffer := make([]byte, 65535)
	n, _, err := radiusConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read Radius message failed: %+v", err)
	}

	// Step 5: 5GNAS
	ueRadiusMessage = new(radius_message.RadiusMessage)
	radiusAuthenticator, err = hex.DecodeString("ea408c3a615fc82899bb8f2fa2e374e9")
	if err != nil {
		fmt.Printf("Failed to decode hex string: %v\n", err)
		return
	}

	ueRadiusMessage.BuildRadiusHeader(radius_message.AccessRequest, 0x06, radiusAuthenticator)
	// create Radius payload
	ueRadiusPayload = new(radius_message.RadiusPayloadContainer)
	*ueRadiusPayload = append(*ueRadiusPayload, *ueUserNamePayload, *calledStationPayload, *callingStationPayload)

	// create EAP message (Expanded) payload
	identifier, err = radius_handler.GenerateRandomUint8()
	if err != nil {
		t.Errorf("Random number failed: %+v", err)
		return
	}
	// EAP-5G vendor type data
	eapVendorTypeData := make([]byte, 2)
	eapVendorTypeData[0] = message.EAP5GType5GNAS
	// AN Parameters
	anParameters := buildEAP5GANParameters()
	anParametersLength := make([]byte, 2)
	binary.BigEndian.PutUint16(anParametersLength, uint16(len(anParameters)))
	eapVendorTypeData = append(eapVendorTypeData, anParametersLength...)
	eapVendorTypeData = append(eapVendorTypeData, anParameters...)

	// NAS-PDU (Registration Request)
	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)

	nasLength := make([]byte, 2)
	binary.BigEndian.PutUint16(nasLength, uint16(len(registrationRequest)))
	eapVendorTypeData = append(eapVendorTypeData, nasLength...)
	eapVendorTypeData = append(eapVendorTypeData, registrationRequest...)

	BuildEAP5GNAS(ueRadiusPayload, identifier, eapVendorTypeData)

	ueRadiusMessage.Payloads = *ueRadiusPayload
	pkt, err = ueRadiusMessage.Encode()
	if err != nil {
		t.Fatalf("Radius Message Encoding error: %+v", err)
	}
	// Send to tngf
	if _, err := radiusConnection.WriteToUDP(pkt, tngfRadiusUDPAddr); err != nil {
		t.Fatalf("Write Radius maessage fail: %+v", err)
	}

	// Step 6: Receive TNGF reply
	buffer = make([]byte, 65535)
	n, _, err = radiusConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read Radius message failed: %+v", err)
	}

	err = ueRadiusMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode Radius message failed: %+v", err)
	}
	var eapMessage []byte

	for _, radiusPayload := range ueRadiusMessage.Payloads {
		switch radiusPayload.Type {
		case radius_message.TypeEAPMessage:
			eapMessage = radiusPayload.Val
		}
	}
	eap := new(radius_message.EAP)
	err = eap.Unmarshal(eapMessage)
	if eap.Code != radius_message.EAPCodeRequest {
		t.Fatalf("[EAP] Received an EAP payload with code other than request. Drop the payload.")
	}

	eapTypeData := eap.EAPTypeData[0]
	var eapExpanded *radius_message.EAPExpanded

	var decodedNAS *nas.Message

	eapExpanded = eapTypeData.(*radius_message.EAPExpanded)

	// Decode NAS - Authentication Request
	nasData := eapExpanded.VendorData[4:]
	decodedNAS = new(nas.Message)
	if err := decodedNAS.PlainNasDecode(&nasData); err != nil {
		t.Fatalf("Decode plain NAS fail: %+v", err)
	}

	// Calculate for RES*
	assert.NotNil(t, decodedNAS)
	rand := decodedNAS.AuthenticationRequest.GetRANDValue()
	resStat := ue.DeriveRESstarAndSetKey(ue.AuthenticationSubs, rand[:], "5G:mnc093.mcc208.3gppnetwork.org")

	// Send Authentication

	ueRadiusMessage = new(radius_message.RadiusMessage)
	radiusAuthenticator, err = hex.DecodeString("ea408c3a615fc82899bb8f2fa2e374e9")

	if err != nil {
		fmt.Printf("Failed to decode hex string: %v\n", err)
		return
	}
	ueRadiusMessage.BuildRadiusHeader(radius_message.AccessRequest, 0x07, radiusAuthenticator)
	ueRadiusPayload = new(radius_message.RadiusPayloadContainer)
	*ueRadiusPayload = append(*ueRadiusPayload, *ueUserNamePayload, *calledStationPayload, *callingStationPayload)
	// create EAP message (Expanded) payload
	identifier, err = radius_handler.GenerateRandomUint8()
	if err != nil {
		t.Errorf("Random number failed: %+v", err)
		return
	}
	// EAP-5G vendor type data
	eapVendorTypeData = make([]byte, 2)
	eapVendorTypeData[0] = message.EAP5GType5GNAS

	// AN Parameters
	eapVendorTypeData = append(eapVendorTypeData, anParametersLength...)
	eapVendorTypeData = append(eapVendorTypeData, anParameters...)

	authenticationResponse := nasTestpacket.GetAuthenticationResponse(resStat, "")
	nasLength = make([]byte, 2)
	binary.BigEndian.PutUint16(nasLength, uint16(len(authenticationResponse)))
	eapVendorTypeData = append(eapVendorTypeData, nasLength...)
	eapVendorTypeData = append(eapVendorTypeData, authenticationResponse...)

	BuildEAP5GNAS(ueRadiusPayload, identifier, eapVendorTypeData)

	ueRadiusMessage.Payloads = *ueRadiusPayload
	pkt, err = ueRadiusMessage.Encode()
	if err != nil {
		t.Fatalf("Radius Message Encoding error: %+v", err)
	}
	// Send to tngf
	if _, err := radiusConnection.WriteToUDP(pkt, tngfRadiusUDPAddr); err != nil {
		t.Fatalf("Write Radius maessage fail: %+v", err)
	}

	// Step 9b: Receive TNGF reply
	buffer = make([]byte, 65535)
	n, _, err = radiusConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read Radius message failed: %+v", err)
	}

	err = ueRadiusMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode Radius message failed: %+v", err)
	}

	// Step 9c:
	ueRadiusMessage = new(radius_message.RadiusMessage)
	radiusAuthenticator, err = hex.DecodeString("ea408c3a615fc82899bb8f2fa2e374e9")
	if err != nil {
		fmt.Printf("Failed to decode hex string: %v\n", err)
		return
	}

	ueRadiusMessage.BuildRadiusHeader(radius_message.AccessRequest, 0x08, radiusAuthenticator)
	// create Radius payload
	ueRadiusPayload = new(radius_message.RadiusPayloadContainer)
	*ueRadiusPayload = append(*ueRadiusPayload, *ueUserNamePayload, *calledStationPayload, *callingStationPayload)

	// create EAP message (Expanded) payload
	identifier, err = radius_handler.GenerateRandomUint8()
	if err != nil {
		t.Errorf("Random number failed: %+v", err)
		return
	}
	// EAP-5G vendor type data
	eapVendorTypeData = make([]byte, 2)
	eapVendorTypeData[0] = message.EAP5GType5GNAS

	// AN Parameters
	anParameters = buildEAP5GANParameters()
	anParametersLength = make([]byte, 2)
	binary.BigEndian.PutUint16(anParametersLength, uint16(len(anParameters)))
	eapVendorTypeData = append(eapVendorTypeData, anParametersLength...)
	eapVendorTypeData = append(eapVendorTypeData, anParameters...)

	// NAS-PDU (SMC Complete)
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	smcComplete := nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
	smcComplete, err = EncodeNasPduWithSecurity(ue, smcComplete, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)
	assert.Nil(t, err)
	nasLength = make([]byte, 2)
	binary.BigEndian.PutUint16(nasLength, uint16(len(smcComplete)))
	eapVendorTypeData = append(eapVendorTypeData, nasLength...)
	eapVendorTypeData = append(eapVendorTypeData, smcComplete...)

	BuildEAP5GNAS(ueRadiusPayload, identifier, eapVendorTypeData)

	ueRadiusMessage.Payloads = *ueRadiusPayload
	pkt, err = ueRadiusMessage.Encode()
	if err != nil {
		t.Fatalf("Radius Message Encoding error: %+v", err)
	}
	// Send to tngf
	if _, err := radiusConnection.WriteToUDP(pkt, tngfRadiusUDPAddr); err != nil {
		t.Fatalf("Write Radius maessage fail: %+v", err)
	}

	// Step 10b: Receive TNGF reply
	buffer = make([]byte, 65535)
	n, _, err = radiusConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read Radius message failed: %+v", err)
	}

	err = ueRadiusMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode Radius message failed: %+v", err)
	}

	// 10c: EAP-Res/5G-Notification
	ueRadiusMessage = new(radius_message.RadiusMessage)
	radiusAuthenticator, err = hex.DecodeString("ea408c3a615fc82899bb8f2fa2e374e9")
	if err != nil {
		fmt.Printf("Failed to decode hex string: %v\n", err)
		return
	}

	ueRadiusMessage.BuildRadiusHeader(radius_message.AccessRequest, 0x09, radiusAuthenticator)
	// create Radius payload
	ueRadiusPayload = new(radius_message.RadiusPayloadContainer)
	*ueRadiusPayload = append(*ueRadiusPayload, *ueUserNamePayload, *calledStationPayload, *callingStationPayload)

	// create EAP message (Expanded) payload
	identifier, err = radius_handler.GenerateRandomUint8()
	if err != nil {
		t.Errorf("Random number failed: %+v", err)
		return
	}
	BuildEAP5GNotification(ueRadiusPayload, identifier)

	ueRadiusMessage.Payloads = *ueRadiusPayload
	pkt, err = ueRadiusMessage.Encode()
	if err != nil {
		t.Fatalf("Radius Message Encoding error: %+v", err)
	}
	// Send to tngf
	if _, err := radiusConnection.WriteToUDP(pkt, tngfRadiusUDPAddr); err != nil {
		t.Fatalf("Write Radius maessage fail: %+v", err)
	}

	// 10e: EAP-Success
	// Receive TNGF reply
	buffer = make([]byte, 65535)
	n, _, err = radiusConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read Radius message failed: %+v", err)
	}

	err = ueRadiusMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode Radius message failed: %+v", err)
	}
	// IKE_SA_INIT
}

// func setUESecurityCapability(ue *RanUeContext) (UESecurityCapability *nasType.UESecurityCapability) {
// 	UESecurityCapability = &nasType.UESecurityCapability{
// 		Iei:    nasMessage.RegistrationRequestUESecurityCapabilityType,
// 		Len:    8,
// 		Buffer: []uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
// 	}
// 	switch ue.CipheringAlg {
// 	case security.AlgCiphering128NEA0:
// 		UESecurityCapability.SetEA0_5G(1)
// 	case security.AlgCiphering128NEA1:
// 		UESecurityCapability.SetEA1_128_5G(1)
// 	case security.AlgCiphering128NEA2:
// 		UESecurityCapability.SetEA2_128_5G(1)
// 	case security.AlgCiphering128NEA3:
// 		UESecurityCapability.SetEA3_128_5G(1)
// 	}

// 	switch ue.IntegrityAlg {
// 	case security.AlgIntegrity128NIA0:
// 		UESecurityCapability.SetIA0_5G(1)
// 	case security.AlgIntegrity128NIA1:
// 		UESecurityCapability.SetIA1_128_5G(1)
// 	case security.AlgIntegrity128NIA2:
// 		UESecurityCapability.SetIA2_128_5G(1)
// 	case security.AlgIntegrity128NIA3:
// 		UESecurityCapability.SetIA3_128_5G(1)
// 	}

// 	return
// }
