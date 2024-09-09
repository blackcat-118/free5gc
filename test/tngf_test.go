package test

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"net"
	"strconv"
	"testing"
	"time"

	"test/nasTestpacket"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-ping/ping"
	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/nasType"
	"github.com/free5gc/nas/security"
	"github.com/free5gc/ngap"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/tngf/pkg/context"
	"github.com/free5gc/tngf/pkg/ike/handler"
	"github.com/free5gc/tngf/pkg/ike/message"
	"github.com/free5gc/tngf/pkg/ike/xfrm"
	radius_handler "github.com/free5gc/tngf/pkg/radius/handler"
	radius_message "github.com/free5gc/tngf/pkg/radius/message"
	"github.com/free5gc/util/ueauth"
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
	tngfUDPAddr, err := net.ResolveUDPAddr("udp", tngfInfo_IPSecIfaceAddr+":500")
	if err != nil {
		t.Fatalf("Resolve UDP address %s fail: %+v", tngfInfo_IPSecIfaceAddr+":500", err)
	}
	// ueUDPAddr, err := net.ResolveUDPAddr("udp", tngfueInfo_IPSecIfaceAddr+":48744")
	// if err != nil {
	// 	t.Fatalf("Resolve UDP address %s fail: %+v", tngfueInfo_IPSecIfaceAddr+":48744", err)
	// }
	udpConnection, err := setupUDPSocket()
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

	// AAA message, send to tngf
	// create a new radius message
	ueRadiusMessage := new(radius_message.RadiusMessage)
	radiusAuthenticator, err := hex.DecodeString("ea408c3a615fc82899bb8f2fa2e374e9")
	if err != nil {
		fmt.Printf("Failed to decode hex string: %v\n", err)
		return
	}

	ueRadiusMessage.BuildRadiusHeader(radius_message.AccessRequest, 0x01, radiusAuthenticator)
	ueRadiusMessage.Length = 0x6c
	// create Radius payload
	ueRadiusPayload := new(radius_message.RadiusPayloadContainer)
	*ueRadiusPayload = append(*ueRadiusPayload, *ueUserNamePayload, *calledStationPayload, *callingStationPayload)

	// Create EAP message (Identity) payload
	identifier, err := radius_handler.GenerateRandomUint8()
	if err != nil {
		t.Errorf("Random number failed: %+v", err)
		return
	}
	BuildEAPIdentity(ueRadiusPayload, identifier, []byte("tngfue"))

	// Create Authenticator payload
	// authPayload := new(radius_message.RadiusPayload)
	// authPayload.Type = radius_message.TypeMessageAuthenticator
	// authPayload.Length = uint8(18)
	// authPayload.Val = make([]byte, 16)
	// ueRadiusMessage.Payloads = *ueRadiusPayload
	// ueRadiusMessage.Payloads = append(ueRadiusMessage.Payloads, *authPayload)
	// authPayload.Val = radius_handler.GetMessageAuthenticator(ueRadiusMessage)
	// *ueRadiusPayload = append(*ueRadiusPayload, *authPayload)

	ueRadiusMessage.Payloads = *ueRadiusPayload
	pkt, err := ueRadiusMessage.Encode()
	if err != nil {
		t.Fatalf("Radius Message Encoding error: %+v", err)
	}
	// Send to tngf
	if _, err := radiusConnection.WriteToUDP(pkt, tngfRadiusUDPAddr); err != nil {
		t.Fatalf("Write Radius maessage fail: %+v", err)
	}
	// radius_handler.SendRadiusMessageToUE(radiusConnection, ueUDPAddr, tngfRadiusUDPAddr, ueRadiusMessage)

	// Receive TNGF reply
	buffer := make([]byte, 65535)
	n, _, err := radiusConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read Radius message failed: %+v", err)
	}

	// 5GNAS
	hexString := "010601145095e4b8e2ac0f2c39bb0e6b0c59a9120108746e676675650406c0a80101200e6434366530653635616361321e1e44342d36452d30452d36352d41432d41323a667265653567632d61703d06000000130606000000020506000000011f1343342d38352d30382d37372d41372d44314d18434f4e4e4543542035344d627073203830322e3131612c1246353238333441373931303734433836ba06000fac04bb06000fac04bc06000fac010c06000005784f4d0236004bfe0028af0000000302000022061077000d0102f839f0ff00000000000070010602f839cafe00040103020302f83900177e004179000d0102f839f0ff000000000000702e02802050129ecce679b9d0b8237e1ef299b7f83f24"
	// Decode the hexadecimal string to bytes
	eapPacket, err := hex.DecodeString(hexString)
	if err != nil {
		fmt.Printf("Failed to decode hex string: %v\n", err)
		return
	}
	// ueRadiusMessage = new(radius_message.RadiusMessage)
	// radiusAuthenticator, err = hex.DecodeString("5095e4b8e2ac0f2c39bb0e6b0c59a912")
	// if err != nil {
	// 	fmt.Printf("Failed to decode hex string: %v\n", err)
	// 	return
	// }

	// ueRadiusMessage.BuildRadiusHeader(radius_message.AccessRequest, 0x06, radiusAuthenticator)

	if _, err := radiusConnection.WriteToUDP(eapPacket, tngfRadiusUDPAddr); err != nil {
		t.Fatalf("Write Radius maessage fail: %+v", err)
	}
	return
	// Receive TNGF reply
	buffer = make([]byte, 65535)
	n, _, err = radiusConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read Radius message failed: %+v", err)
	}
	// IKE_SA_INIT
	ikeInitiatorSPI := uint64(123123)
	ikeMessage := new(message.IKEMessage)
	ikeMessage.BuildIKEHeader(ikeInitiatorSPI, 0, message.IKE_SA_INIT, message.InitiatorBitCheck, 0)

	// Security Association
	securityAssociation := ikeMessage.Payloads.BuildSecurityAssociation()
	// Proposal 1
	proposal := securityAssociation.Proposals.BuildProposal(1, message.TypeIKE, nil)
	// ENCR
	var attributeType uint16 = message.AttributeTypeKeyLength
	var keyLength uint16 = 256
	proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_AES_CBC, &attributeType, &keyLength, nil)
	// INTEG
	proposal.IntegrityAlgorithm.BuildTransform(message.TypeIntegrityAlgorithm, message.AUTH_HMAC_SHA1_96, nil, nil, nil)
	// PRF
	proposal.PseudorandomFunction.BuildTransform(message.TypePseudorandomFunction, message.PRF_HMAC_SHA1, nil, nil, nil)
	// DH
	proposal.DiffieHellmanGroup.BuildTransform(message.TypeDiffieHellmanGroup, message.DH_2048_BIT_MODP, nil, nil, nil)

	// Key exchange data
	generator := new(big.Int).SetUint64(handler.Group14Generator)
	factor, ok := new(big.Int).SetString(handler.Group14PrimeString, 16)
	if !ok {
		t.Fatalf("Generate key exchange data failed")
	}
	secert := handler.GenerateRandomNumber()
	localPublicKeyExchangeValue := new(big.Int).Exp(generator, secert, factor).Bytes()
	prependZero := make([]byte, len(factor.Bytes())-len(localPublicKeyExchangeValue))
	localPublicKeyExchangeValue = append(prependZero, localPublicKeyExchangeValue...)
	ikeMessage.Payloads.BUildKeyExchange(message.DH_2048_BIT_MODP, localPublicKeyExchangeValue)

	// Nonce
	localNonce := handler.GenerateRandomNumber().Bytes()
	ikeMessage.Payloads.BuildNonce(localNonce)

	// Send to TNGF
	ikeMessageData, err := ikeMessage.Encode()
	if err != nil {
		t.Fatalf("Encode IKE Message fail: %+v", err)
	}
	if _, err := udpConnection.WriteToUDP(ikeMessageData, tngfUDPAddr); err != nil {
		t.Fatalf("Write IKE maessage fail: %+v", err)
	}
	realMessage1, _ := ikeMessage.Encode()
	ikeSecurityAssociation := &context.IKESecurityAssociation{
		ResponderSignedOctets: realMessage1,
	}

	// Receive TNGF reply
	n, _, err = udpConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read IKE Message fail: %+v", err)
	}
	ikeMessage.Payloads.Reset()
	err = ikeMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode IKE Message fail: %+v", err)
	}

	var sharedKeyExchangeData []byte
	var remoteNonce []byte

	for _, ikePayload := range ikeMessage.Payloads {
		switch ikePayload.Type() {
		case message.TypeSA:
			t.Log("Get SA payload")
		case message.TypeKE:
			remotePublicKeyExchangeValue := ikePayload.(*message.KeyExchange).KeyExchangeData
			var i int = 0
			for {
				if remotePublicKeyExchangeValue[i] != 0 {
					break
				}
			}
			remotePublicKeyExchangeValue = remotePublicKeyExchangeValue[i:]
			remotePublicKeyExchangeValueBig := new(big.Int).SetBytes(remotePublicKeyExchangeValue)
			sharedKeyExchangeData = new(big.Int).Exp(remotePublicKeyExchangeValueBig, secert, factor).Bytes()
		case message.TypeNiNr:
			remoteNonce = ikePayload.(*message.Nonce).NonceData
		}
	}

	ikeSecurityAssociation = &context.IKESecurityAssociation{
		LocalSPI:               ikeInitiatorSPI,
		RemoteSPI:              ikeMessage.ResponderSPI,
		InitiatorMessageID:     0,
		ResponderMessageID:     0,
		EncryptionAlgorithm:    proposal.EncryptionAlgorithm[0],
		IntegrityAlgorithm:     proposal.IntegrityAlgorithm[0],
		PseudorandomFunction:   proposal.PseudorandomFunction[0],
		DiffieHellmanGroup:     proposal.DiffieHellmanGroup[0],
		ConcatenatedNonce:      append(localNonce, remoteNonce...),
		DiffieHellmanSharedKey: sharedKeyExchangeData,
		ResponderSignedOctets:  append(ikeSecurityAssociation.ResponderSignedOctets, remoteNonce...),
	}

	if err := tngfgenerateKeyForIKESA(ikeSecurityAssociation); err != nil {
		t.Fatalf("Generate key for IKE SA failed: %+v", err)
	}

	tngfue.TNGFIKESecurityAssociation = ikeSecurityAssociation

	// IKE_AUTH
	ikeMessage.Payloads.Reset()
	tngfue.TNGFIKESecurityAssociation.InitiatorMessageID++
	ikeMessage.BuildIKEHeader(
		tngfue.TNGFIKESecurityAssociation.LocalSPI, tngfue.TNGFIKESecurityAssociation.RemoteSPI,
		message.IKE_AUTH, message.InitiatorBitCheck, tngfue.TNGFIKESecurityAssociation.InitiatorMessageID)

	var ikePayload message.IKEPayloadContainer

	// Identification
	ikePayload.BuildIdentificationInitiator(message.ID_KEY_ID, []byte("UE"))

	// Security Association
	securityAssociation = ikePayload.BuildSecurityAssociation()
	// Proposal 1
	inboundSPI := tngfgenerateSPI(tngfue)
	proposal = securityAssociation.Proposals.BuildProposal(1, message.TypeESP, inboundSPI)
	// ENCR
	proposal.EncryptionAlgorithm.BuildTransform(message.TypeEncryptionAlgorithm, message.ENCR_AES_CBC, &attributeType, &keyLength, nil)
	// INTEG
	proposal.IntegrityAlgorithm.BuildTransform(message.TypeIntegrityAlgorithm, message.AUTH_HMAC_SHA1_96, nil, nil, nil)
	// ESN
	proposal.ExtendedSequenceNumbers.BuildTransform(message.TypeExtendedSequenceNumbers, message.ESN_NO, nil, nil, nil)

	// Traffic Selector
	tsi := ikePayload.BuildTrafficSelectorInitiator()
	tsi.TrafficSelectors.BuildIndividualTrafficSelector(message.TS_IPV4_ADDR_RANGE, 0, 0, 65535, []byte{0, 0, 0, 0}, []byte{255, 255, 255, 255})
	tsr := ikePayload.BuildTrafficSelectorResponder()
	tsr.TrafficSelectors.BuildIndividualTrafficSelector(message.TS_IPV4_ADDR_RANGE, 0, 0, 65535, []byte{0, 0, 0, 0}, []byte{255, 255, 255, 255})

	if err := tngfencryptProcedure(ikeSecurityAssociation, ikePayload, ikeMessage); err != nil {
		t.Fatalf("Encrypting IKE message failed: %+v", err)
	}

	// Send to TNGF
	ikeMessageData, err = ikeMessage.Encode()
	if err != nil {
		t.Fatalf("Encode IKE message failed: %+v", err)
	}
	if _, err := udpConnection.WriteToUDP(ikeMessageData, tngfUDPAddr); err != nil {
		t.Fatalf("Write IKE message failed: %+v", err)
	}

	tngfue.CreateHalfChildSA(tngfue.TNGFIKESecurityAssociation.InitiatorMessageID, binary.BigEndian.Uint32(inboundSPI), -1)

	// Receive TNGF reply
	n, _, err = udpConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read IKE message failed: %+v", err)
	}
	ikeMessage.Payloads.Reset()
	err = ikeMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode IKE message failed: %+v", err)
	}

	encryptedPayload, ok := ikeMessage.Payloads[0].(*message.Encrypted)
	if !ok {
		t.Fatalf("Received payload is not an encrypted payload")
	}

	decryptedIKEPayload, err := tngfdecryptProcedure(ikeSecurityAssociation, ikeMessage, encryptedPayload)
	if err != nil {
		t.Fatalf("Decrypt IKE message failed: %+v", err)
	}

	var eapIdentifier uint8

	for _, ikePayload := range decryptedIKEPayload {
		switch ikePayload.Type() {
		case message.TypeIDr:
			t.Log("Get IDr")
		case message.TypeAUTH:
			t.Log("Get AUTH")
		case message.TypeCERT:
			t.Log("Get CERT")
		case message.TypeEAP:
			eapIdentifier = ikePayload.(*message.EAP).Identifier
			t.Log("Get EAP")
		}
	}

	// IKE_AUTH - EAP exchange
	ikeMessage.Payloads.Reset()
	tngfue.TNGFIKESecurityAssociation.InitiatorMessageID++
	ikeMessage.BuildIKEHeader(tngfue.TNGFIKESecurityAssociation.LocalSPI, tngfue.TNGFIKESecurityAssociation.RemoteSPI,
		message.IKE_AUTH, message.InitiatorBitCheck, tngfue.TNGFIKESecurityAssociation.InitiatorMessageID)

	ikePayload.Reset()

	// EAP-5G vendor type data
	eapVendorTypeData := make([]byte, 2)
	eapVendorTypeData[0] = message.EAP5GType5GNAS

	// AN Parameters
	anParameters := buildEAP5GANParameters()
	anParametersLength := make([]byte, 2)
	binary.BigEndian.PutUint16(anParametersLength, uint16(len(anParameters)))
	eapVendorTypeData = append(eapVendorTypeData, anParametersLength...)
	eapVendorTypeData = append(eapVendorTypeData, anParameters...)

	// NAS
	ueSecurityCapability := ue.GetUESecurityCapability()
	registrationRequest := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, nil, nil, nil)

	nasLength := make([]byte, 2)
	binary.BigEndian.PutUint16(nasLength, uint16(len(registrationRequest)))
	eapVendorTypeData = append(eapVendorTypeData, nasLength...)
	eapVendorTypeData = append(eapVendorTypeData, registrationRequest...)

	eap := ikePayload.BuildEAP(message.EAPCodeResponse, eapIdentifier)
	eap.EAPTypeData.BuildEAPExpanded(message.VendorID3GPP, message.VendorTypeEAP5G, eapVendorTypeData)

	if err := tngfencryptProcedure(ikeSecurityAssociation, ikePayload, ikeMessage); err != nil {
		t.Fatalf("Encrypt IKE message failed: %+v", err)
	}

	// Send to TNGF
	ikeMessageData, err = ikeMessage.Encode()
	if err != nil {
		t.Fatalf("Encode IKE message failed: %+v", err)
	}
	if _, err := udpConnection.WriteToUDP(ikeMessageData, tngfUDPAddr); err != nil {
		t.Fatalf("Write IKE message failed: %+v", err)
	}

	// Receive TNGF reply
	n, _, err = udpConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read IKE message failed: %+v", err)
	}

	ikeMessage.Payloads.Reset()
	err = ikeMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode IKE message failed: %+v", err)
	}

	encryptedPayload, ok = ikeMessage.Payloads[0].(*message.Encrypted)
	if !ok {
		t.Fatalf("Received payload is not an encrypted payload")
	}

	decryptedIKEPayload, err = tngfdecryptProcedure(ikeSecurityAssociation, ikeMessage, encryptedPayload)
	if err != nil {
		t.Fatalf("Decrypt IKE message failed: %+v", err)
	}

	var eapReq *message.EAP
	var eapExpanded *message.EAPExpanded

	eapReq, ok = decryptedIKEPayload[0].(*message.EAP)
	if !ok {
		t.Fatalf("Received packet is not an EAP payload")
	}

	var decodedNAS *nas.Message

	eapExpanded, ok = eapReq.EAPTypeData[0].(*message.EAPExpanded)
	if !ok {
		t.Fatalf("The EAP data is not an EAP expended.")
	}

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

	// send NAS Authentication Response
	pdu := nasTestpacket.GetAuthenticationResponse(resStat, "")

	// IKE_AUTH - EAP exchange
	ikeMessage.Payloads.Reset()
	tngfue.TNGFIKESecurityAssociation.InitiatorMessageID++
	ikeMessage.BuildIKEHeader(tngfue.TNGFIKESecurityAssociation.LocalSPI, tngfue.TNGFIKESecurityAssociation.RemoteSPI,
		message.IKE_AUTH, message.InitiatorBitCheck, tngfue.TNGFIKESecurityAssociation.InitiatorMessageID)

	ikePayload.Reset()

	// EAP-5G vendor type data
	eapVendorTypeData = make([]byte, 4)
	eapVendorTypeData[0] = message.EAP5GType5GNAS

	// NAS - Authentication Response
	nasLength = make([]byte, 2)
	binary.BigEndian.PutUint16(nasLength, uint16(len(pdu)))
	eapVendorTypeData = append(eapVendorTypeData, nasLength...)
	eapVendorTypeData = append(eapVendorTypeData, pdu...)

	eap = ikePayload.BuildEAP(message.EAPCodeResponse, eapReq.Identifier)
	eap.EAPTypeData.BuildEAPExpanded(message.VendorID3GPP, message.VendorTypeEAP5G, eapVendorTypeData)

	err = tngfencryptProcedure(ikeSecurityAssociation, ikePayload, ikeMessage)
	if err != nil {
		t.Fatalf("Encrypt IKE message failed: %+v", err)
	}

	// Send to TNGF
	ikeMessageData, err = ikeMessage.Encode()
	if err != nil {
		t.Fatalf("Encode IKE Message fail: %+v", err)
	}
	_, err = udpConnection.WriteToUDP(ikeMessageData, tngfUDPAddr)
	if err != nil {
		t.Fatalf("Write IKE message failed: %+v", err)
	}

	// Receive TNGF reply
	n, _, err = udpConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read IKE Message fail: %+v", err)
	}
	ikeMessage.Payloads.Reset()
	err = ikeMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode IKE Message fail: %+v", err)
	}
	encryptedPayload, ok = ikeMessage.Payloads[0].(*message.Encrypted)
	if !ok {
		t.Fatal("Received pakcet is not an encrypted payload")
	}
	decryptedIKEPayload, err = tngfdecryptProcedure(ikeSecurityAssociation, ikeMessage, encryptedPayload)
	if err != nil {
		t.Fatalf("Decrypt IKE message failed: %+v", err)
		return
	}
	eapReq, ok = decryptedIKEPayload[0].(*message.EAP)
	if !ok {
		t.Fatal("Received packet is not an EAP payload")
		return
	}
	eapExpanded, ok = eapReq.EAPTypeData[0].(*message.EAPExpanded)
	if !ok {
		t.Fatal("Received packet is not an EAP expended payload")
		return
	}

	nasData = eapExpanded.VendorData[4:]

	// Send NAS Security Mode Complete Msg
	registrationRequestWith5GMM := nasTestpacket.GetRegistrationRequest(nasMessage.RegistrationType5GSInitialRegistration,
		mobileIdentity5GS, nil, ueSecurityCapability, ue.Get5GMMCapability(), nil, nil)
	pdu = nasTestpacket.GetSecurityModeComplete(registrationRequestWith5GMM)
	pdu, err = EncodeNasPduWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCipheredWithNew5gNasSecurityContext, true, true)
	assert.Nil(t, err)

	// IKE_AUTH - EAP exchange
	ikeMessage.Payloads.Reset()
	tngfue.TNGFIKESecurityAssociation.InitiatorMessageID++
	ikeMessage.BuildIKEHeader(tngfue.TNGFIKESecurityAssociation.LocalSPI, tngfue.TNGFIKESecurityAssociation.RemoteSPI,
		message.IKE_AUTH, message.InitiatorBitCheck, tngfue.TNGFIKESecurityAssociation.InitiatorMessageID)

	ikePayload.Reset()

	// EAP-5G vendor type data
	eapVendorTypeData = make([]byte, 4)
	eapVendorTypeData[0] = message.EAP5GType5GNAS

	// NAS - Authentication Response
	nasLength = make([]byte, 2)
	binary.BigEndian.PutUint16(nasLength, uint16(len(pdu)))
	eapVendorTypeData = append(eapVendorTypeData, nasLength...)
	eapVendorTypeData = append(eapVendorTypeData, pdu...)

	eap = ikePayload.BuildEAP(message.EAPCodeResponse, eapReq.Identifier)
	eap.EAPTypeData.BuildEAPExpanded(message.VendorID3GPP, message.VendorTypeEAP5G, eapVendorTypeData)

	err = tngfencryptProcedure(ikeSecurityAssociation, ikePayload, ikeMessage)
	if err != nil {
		t.Fatalf("Encrypt IKE message failed: %+v", err)
	}

	// Send to TNGF
	ikeMessageData, err = ikeMessage.Encode()
	if err != nil {
		t.Fatalf("Encode IKE Message fail: %+v", err)
	}
	_, err = udpConnection.WriteToUDP(ikeMessageData, tngfUDPAddr)
	if err != nil {
		t.Fatalf("Write IKE message failed: %+v", err)
	}

	// Receive TNGF reply
	n, _, err = udpConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read IKE Message fail: %+v", err)
		return
	}
	ikeMessage.Payloads.Reset()
	err = ikeMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode IKE Message fail: %+v", err)
	}
	encryptedPayload, ok = ikeMessage.Payloads[0].(*message.Encrypted)
	if !ok {
		t.Fatal("Received pakcet is not an encrypted payload")
	}
	decryptedIKEPayload, err = tngfdecryptProcedure(ikeSecurityAssociation, ikeMessage, encryptedPayload)
	if err != nil {
		t.Fatal(err)
	}
	eapReq, ok = decryptedIKEPayload[0].(*message.EAP)
	if !ok {
		t.Fatal("Received packet is not an EAP payload")
	}
	if eapReq.Code != message.EAPCodeSuccess {
		t.Fatal("Not Success")
	}

	// IKE_AUTH - Authentication
	ikeMessage.Payloads.Reset()
	tngfue.TNGFIKESecurityAssociation.InitiatorMessageID++
	ikeMessage.BuildIKEHeader(tngfue.TNGFIKESecurityAssociation.LocalSPI, tngfue.TNGFIKESecurityAssociation.RemoteSPI,
		message.IKE_AUTH, message.InitiatorBitCheck, tngfue.TNGFIKESecurityAssociation.InitiatorMessageID)

	ikePayload.Reset()

	// Authentication
	// Derive Ktngf
	P0 := make([]byte, 4)
	binary.BigEndian.PutUint32(P0, ue.ULCount.Get()-1)
	L0 := ueauth.KDFLen(P0)
	P1 := []byte{security.AccessTypeNon3GPP}
	L1 := ueauth.KDFLen(P1)

	Ktngf, err := ueauth.GetKDFValue(ue.Kamf, ueauth.FC_FOR_KGNB_KN3IWF_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		t.Fatalf("Get Ktngf error : %+v", err)
	}

	pseudorandomFunction, ok := handler.NewPseudorandomFunction(ikeSecurityAssociation.SK_pi,
		ikeSecurityAssociation.PseudorandomFunction.TransformID)
	if !ok {
		t.Fatalf("Get an unsupported pseudorandom funcion. This may imply an unsupported transform is chosen.")
	}
	var idPayload message.IKEPayloadContainer
	idPayload.BuildIdentificationInitiator(message.ID_KEY_ID, []byte("UE"))
	idPayloadData, err := idPayload.Encode()
	if err != nil {
		t.Fatalf("Encode IKE payload failed : %+v", err)
	}
	if _, err := pseudorandomFunction.Write(idPayloadData[4:]); err != nil {
		t.Fatalf("Pseudorandom function write error: %+v", err)
	}
	ikeSecurityAssociation.ResponderSignedOctets = append(ikeSecurityAssociation.ResponderSignedOctets,
		pseudorandomFunction.Sum(nil)...)

	transformPseudorandomFunction := ikeSecurityAssociation.PseudorandomFunction

	pseudorandomFunction, ok = handler.NewPseudorandomFunction(Ktngf, transformPseudorandomFunction.TransformID)
	if !ok {
		t.Fatalf("Get an unsupported pseudorandom funcion. This may imply an unsupported transform is chosen.")
	}
	if _, err := pseudorandomFunction.Write([]byte("Key Pad for IKEv2")); err != nil {
		t.Fatalf("Pseudorandom function write error: %+v", err)
	}
	secret := pseudorandomFunction.Sum(nil)
	pseudorandomFunction, ok = handler.NewPseudorandomFunction(secret, transformPseudorandomFunction.TransformID)
	if !ok {
		t.Fatalf("Get an unsupported pseudorandom funcion. This may imply an unsupported transform is chosen.")
	}
	pseudorandomFunction.Reset()
	if _, err := pseudorandomFunction.Write(ikeSecurityAssociation.ResponderSignedOctets); err != nil {
		t.Fatalf("Pseudorandom function write error: %+v", err)
	}
	ikePayload.BuildAuthentication(message.SharedKeyMesageIntegrityCode, pseudorandomFunction.Sum(nil))

	// Configuration Request
	configurationRequest := ikePayload.BuildConfiguration(message.CFG_REQUEST)
	configurationRequest.ConfigurationAttribute.BuildConfigurationAttribute(message.INTERNAL_IP4_ADDRESS, nil)

	err = tngfencryptProcedure(ikeSecurityAssociation, ikePayload, ikeMessage)
	if err != nil {
		t.Fatalf("Encrypt IKE message failed: %+v", err)
	}

	// Send to TNGF
	ikeMessageData, err = ikeMessage.Encode()
	if err != nil {
		t.Fatalf("Encode IKE Message fail: %+v", err)
	}
	_, err = udpConnection.WriteToUDP(ikeMessageData, tngfUDPAddr)
	if err != nil {
		t.Fatalf("Write IKE message failed: %+v", err)
	}

	// Receive TNGF reply
	n, _, err = udpConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read IKE Message fail: %+v", err)
	}
	ikeMessage.Payloads.Reset()
	err = ikeMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode IKE Message fail: %+v", err)
	}
	encryptedPayload, ok = ikeMessage.Payloads[0].(*message.Encrypted)
	if !ok {
		t.Fatal("Received pakcet is not an encrypted payload")
	}
	decryptedIKEPayload, err = tngfdecryptProcedure(ikeSecurityAssociation, ikeMessage, encryptedPayload)
	if err != nil {
		t.Fatal(err)

	}

	// AUTH, SAr2, TSi, Tsr, N(NAS_IP_ADDRESS), N(NAS_TCP_PORT)
	var responseSecurityAssociation *message.SecurityAssociation
	var responseTrafficSelectorInitiator *message.TrafficSelectorInitiator
	var responseTrafficSelectorResponder *message.TrafficSelectorResponder
	var responseConfiguration *message.Configuration
	tngfNASAddr := new(net.TCPAddr)

	for _, ikePayload := range decryptedIKEPayload {
		switch ikePayload.Type() {
		case message.TypeAUTH:
			t.Log("Get Authentication from TNGF")
		case message.TypeSA:
			responseSecurityAssociation = ikePayload.(*message.SecurityAssociation)
			tngfue.TNGFIKESecurityAssociation.IKEAuthResponseSA = responseSecurityAssociation
		case message.TypeTSi:
			responseTrafficSelectorInitiator = ikePayload.(*message.TrafficSelectorInitiator)
		case message.TypeTSr:
			responseTrafficSelectorResponder = ikePayload.(*message.TrafficSelectorResponder)
		case message.TypeN:
			notification := ikePayload.(*message.Notification)
			if notification.NotifyMessageType == message.Vendor3GPPNotifyTypeNAS_IP4_ADDRESS {
				tngfNASAddr.IP = net.IPv4(notification.NotificationData[0], notification.NotificationData[1], notification.NotificationData[2], notification.NotificationData[3])
			}
			if notification.NotifyMessageType == message.Vendor3GPPNotifyTypeNAS_TCP_PORT {
				tngfNASAddr.Port = int(binary.BigEndian.Uint16(notification.NotificationData))
			}
		case message.TypeCP:
			responseConfiguration = ikePayload.(*message.Configuration)
			if responseConfiguration.ConfigurationType == message.CFG_REPLY {
				for _, configAttr := range responseConfiguration.ConfigurationAttribute {
					if configAttr.Type == message.INTERNAL_IP4_ADDRESS {
						tngfueInnerAddr.IP = configAttr.Value
					}
					if configAttr.Type == message.INTERNAL_IP4_NETMASK {
						tngfueInnerAddr.Mask = configAttr.Value
					}
				}
			}
		}
	}

	OutboundSPI := binary.BigEndian.Uint32(tngfue.TNGFIKESecurityAssociation.IKEAuthResponseSA.Proposals[0].SPI)
	childSecurityAssociationContext, err := tngfue.CompleteChildSA(
		0x01, OutboundSPI, tngfue.TNGFIKESecurityAssociation.IKEAuthResponseSA)
	if err != nil {
		t.Fatalf("Create child security association context failed: %+v", err)
	}
	err = tngfparseIPAddressInformationToChildSecurityAssociation(childSecurityAssociationContext,
		responseTrafficSelectorInitiator.TrafficSelectors[0],
		responseTrafficSelectorResponder.TrafficSelectors[0])

	if err != nil {
		t.Fatalf("Parse IP address to child security association failed: %+v", err)
	}
	// Select TCP traffic
	childSecurityAssociationContext.SelectedIPProtocol = unix.IPPROTO_TCP

	if err := tngfgenerateKeyForChildSA(ikeSecurityAssociation, childSecurityAssociationContext); err != nil {
		t.Fatalf("Generate key for child SA failed: %+v", err)
	}

	var linkIPSec netlink.Link

	// Setup interface for ipsec
	newXfrmiName := fmt.Sprintf("%s-default", tngfueInfo_XfrmiName)
	if linkIPSec, err = setupIPsecXfrmi(newXfrmiName, tngfueInfo_IPSecIfaceName, tngfueInfo_XfrmiId, tngfueInnerAddr); err != nil {
		t.Fatalf("Setup XFRM interface %s fail: %+v", newXfrmiName, err)
	}

	defer func() {
		if err := netlink.LinkDel(linkIPSec); err != nil {
			t.Fatalf("Delete XFRM interface %s fail: %+v", newXfrmiName, err)
		} else {
			t.Logf("Delete XFRM interface: %s", newXfrmiName)
		}
	}()

	// Aplly XFRM rules
	if err = tngfapplyXFRMRule(true, tngfueInfo_XfrmiId, childSecurityAssociationContext); err != nil {
		t.Fatalf("Applying XFRM rules failed: %+v", err)
	}

	defer func() {
		_ = netlink.XfrmPolicyFlush()
		_ = netlink.XfrmStateFlush(netlink.XFRM_PROTO_IPSEC_ANY)
	}()

	localTCPAddr := &net.TCPAddr{
		IP: tngfueInnerAddr.IP,
	}
	tcpConnWithTNGF, err := net.DialTCP("tcp", localTCPAddr, tngfNASAddr)
	if err != nil {
		t.Fatal(err)
	}

	nasEnv := make([]byte, 65535)

	n, err = tcpConnWithTNGF.Read(nasEnv)
	if err != nil {
		t.Fatal(err)
		return
	}

	nasEnv, n, err = DecapNasPduFromEnvelope(nasEnv[:n])
	if err != nil {
		t.Fatal(err)
	}
	nasMsg, err := NASDecode(ue, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, nasEnv[:n])
	if err != nil {
		t.Fatalf("NAS Decode Fail: %+v", err)
	}

	spew.Config.Indent = "\t"
	nasStr := spew.Sdump(nasMsg)
	t.Logf("Get NAS Security Mode Command Message:\n %+v", nasStr)

	// send NAS Registration Complete Msg
	pdu = nasTestpacket.GetRegistrationComplete(nil)
	pdu, err = EncodeNasPduInEnvelopeWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		t.Fatal(err)
		return
	}
	_, err = tcpConnWithTNGF.Write(pdu)
	if err != nil {
		t.Fatal(err)
		return
	}

	time.Sleep(500 * time.Millisecond)

	// UE request PDU session setup
	sNssai := models.Snssai{
		Sst: 1,
		Sd:  "fedcba",
	}

	var pduSessionId uint8 = 1

	pdu = nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(pduSessionId, nasMessage.ULNASTransportRequestTypeInitialRequest, "internet", &sNssai)
	pdu, err = EncodeNasPduInEnvelopeWithSecurity(ue, pdu, nas.SecurityHeaderTypeIntegrityProtectedAndCiphered, true, false)
	if err != nil {
		t.Fatal(err)
		return
	}
	_, err = tcpConnWithTNGF.Write(pdu)
	if err != nil {
		t.Fatal(err)
		return
	}

	// Receive TNGF reply
	n, _, err = udpConnection.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Read IKE Message fail: %+v", err)
	}
	ikeMessage.Payloads.Reset()
	err = ikeMessage.Decode(buffer[:n])
	if err != nil {
		t.Fatalf("Decode IKE Message fail: %+v", err)
	}
	t.Logf("IKE message exchange type: %d", ikeMessage.ExchangeType)
	t.Logf("IKE message ID: %d", ikeMessage.MessageID)
	encryptedPayload, ok = ikeMessage.Payloads[0].(*message.Encrypted)
	if !ok {
		t.Fatal("Received pakcet is not an encrypted payload")
		return
	}
	decryptedIKEPayload, err = tngfdecryptProcedure(ikeSecurityAssociation, ikeMessage, encryptedPayload)
	if err != nil {
		t.Fatal(err)
		return
	}

	var QoSInfo *PDUQoSInfo

	var upIPAddr net.IP
	for _, ikePayload := range decryptedIKEPayload {
		switch ikePayload.Type() {
		case message.TypeSA:
			responseSecurityAssociation = ikePayload.(*message.SecurityAssociation)
			OutboundSPI = binary.BigEndian.Uint32(responseSecurityAssociation.Proposals[0].SPI)
		case message.TypeTSi:
			responseTrafficSelectorInitiator = ikePayload.(*message.TrafficSelectorInitiator)
		case message.TypeTSr:
			responseTrafficSelectorResponder = ikePayload.(*message.TrafficSelectorResponder)
		case message.TypeN:
			notification := ikePayload.(*message.Notification)
			if notification.NotifyMessageType == message.Vendor3GPPNotifyType5G_QOS_INFO {
				t.Log("Received Qos Flow settings")
				if info, err := tngfparse5GQoSInfoNotify(notification); err == nil {
					QoSInfo = info
					t.Logf("NotificationData:%+v", notification.NotificationData)
					if QoSInfo.isDSCPSpecified {
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
			ikeSecurityAssociation.ConcatenatedNonce = responseNonce.NonceData
		}
	}

	// IKE CREATE_CHILD_SA response
	ikeMessage.Payloads.Reset()
	tngfue.TNGFIKESecurityAssociation.ResponderMessageID = ikeMessage.MessageID
	ikeMessage.BuildIKEHeader(ikeMessage.InitiatorSPI, ikeMessage.ResponderSPI, message.CREATE_CHILD_SA,
		message.ResponseBitCheck|message.InitiatorBitCheck, tngfue.TNGFIKESecurityAssociation.ResponderMessageID)

	ikePayload.Reset()

	// SA
	inboundSPI = tngfgenerateSPI(tngfue)
	responseSecurityAssociation.Proposals[0].SPI = inboundSPI
	ikePayload = append(ikePayload, responseSecurityAssociation)

	// TSi
	ikePayload = append(ikePayload, responseTrafficSelectorInitiator)

	// TSr
	ikePayload = append(ikePayload, responseTrafficSelectorResponder)

	// Nonce
	localNonce = handler.GenerateRandomNumber().Bytes()
	ikeSecurityAssociation.ConcatenatedNonce = append(ikeSecurityAssociation.ConcatenatedNonce, localNonce...)
	ikePayload.BuildNonce(localNonce)

	if err := tngfencryptProcedure(ikeSecurityAssociation, ikePayload, ikeMessage); err != nil {
		t.Fatalf("Encrypt IKE message failed: %+v", err)
	}

	// Send to TNGF
	ikeMessageData, err = ikeMessage.Encode()
	if err != nil {
		t.Fatalf("Encode IKE Message fail: %+v", err)
	}
	_, err = udpConnection.WriteToUDP(ikeMessageData, tngfUDPAddr)
	if err != nil {
		t.Fatalf("Write IKE message failed: %+v", err)
	}

	tngfue.CreateHalfChildSA(tngfue.TNGFIKESecurityAssociation.ResponderMessageID, binary.BigEndian.Uint32(inboundSPI), -1)
	childSecurityAssociationContextUserPlane, err := tngfue.CompleteChildSA(
		tngfue.TNGFIKESecurityAssociation.ResponderMessageID, OutboundSPI, responseSecurityAssociation)

	if err != nil {
		t.Fatalf("Create child security association context failed: %+v", err)
	}
	err = tngfparseIPAddressInformationToChildSecurityAssociation(childSecurityAssociationContextUserPlane, responseTrafficSelectorResponder.TrafficSelectors[0], responseTrafficSelectorInitiator.TrafficSelectors[0])
	if err != nil {
		t.Fatalf("Parse IP address to child security association failed: %+v", err)
	}
	// Select GRE traffic
	childSecurityAssociationContextUserPlane.SelectedIPProtocol = unix.IPPROTO_GRE

	if err := tngfgenerateKeyForChildSA(ikeSecurityAssociation, childSecurityAssociationContextUserPlane); err != nil {
		t.Fatalf("Generate key for child SA failed: %+v", err)
	}

	// Aplly XFRM rules
	if err = tngfapplyXFRMRule(false, tngfueInfo_XfrmiId, childSecurityAssociationContextUserPlane); err != nil {
		t.Fatalf("Applying XFRM rules failed: %+v", err)
	}

	// TODO
	// We don't check any of message in UeConfigUpdate Message
	if n, err := tcpConnWithTNGF.Read(buffer); err != nil {
		t.Fatalf("No UeConfigUpdate Message: %+v", err)
		_, err := ngap.Decoder(buffer[2:n])
		if err != nil {
			t.Fatalf("UeConfigUpdate Decode Error: %+v", err)
		}
	}

	var pduAddress net.IP

	// Read NAS from TNGF
	if n, err := tcpConnWithTNGF.Read(buffer); err != nil {
		t.Fatalf("Read NAS Message Fail:%+v", err)
	} else {
		nasMsg, err := DecodePDUSessionEstablishmentAccept(ue, n, buffer)
		if err != nil {
			t.Fatalf("DecodePDUSessionEstablishmentAccept Fail: %+v", err)
		}

		spew.Config.Indent = "\t"
		nasStr := spew.Sdump(nasMsg)
		t.Log("Dump DecodePDUSessionEstablishmentAccept:\n", nasStr)
		pduAddress, err = GetPDUAddress(nasMsg.GsmMessage.PDUSessionEstablishmentAccept)
		if err != nil {
			t.Fatalf("GetPDUAddress Fail: %+v", err)
		}

		t.Logf("PDU Address: %s", pduAddress.String())
	}

	var linkGRE netlink.Link

	newGREName := fmt.Sprintf("%s-id-%d", tngfueInfo_GreIfaceName, tngfueInfo_XfrmiId)

	if linkGRE, err = setupGreTunnel(newGREName, newXfrmiName, tngfueInnerAddr.IP, upIPAddr, pduAddress, QoSInfo, t); err != nil {
		t.Fatalf("Setup GRE tunnel %s Fail %+v", newGREName, err)
	}

	defer func() {
		_ = netlink.LinkDel(linkGRE)
		t.Logf("Delete interface: %s", linkGRE.Attrs().Name)
	}()

	// Add route
	upRoute := &netlink.Route{
		LinkIndex: linkGRE.Attrs().Index,
		Dst: &net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.IPv4Mask(0, 0, 0, 0),
		},
	}
	if err := netlink.RouteAdd(upRoute); err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 3; i++ {
		var (
			ifaces []netlink.Link
			err    error
		)
		t.Logf("%d times PDU Session Est Request Start", i+1)
		if ifaces, err = tngfsendPduSessionEstablishmentRequest(pduSessionId+uint8(i), ue, tngfue, ikeSecurityAssociation, udpConnection, tcpConnWithTNGF, t); err != nil {
			t.Fatalf("Session Est Request Fail: %+v", err)
		} else {
			t.Logf("Create %d interfaces", len(ifaces))
		}

		defer func() {
			for _, iface := range ifaces {
				if err := netlink.LinkDel(iface); err != nil {
					t.Fatalf("Delete interface %s fail: %+v", iface.Attrs().Name, err)
				} else {
					t.Logf("Delete interface: %s", iface.Attrs().Name)
				}
			}
		}()
	}

	// Ping remote
	pinger, err := ping.NewPinger("10.60.0.101")
	if err != nil {
		t.Fatal(err)
		return
	}

	// Run with root
	pinger.SetPrivileged(true)

	pinger.OnRecv = func(pkt *ping.Packet) {
		t.Logf("%d bytes from %s: icmp_seq=%d time=%v\n",
			pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
	}
	pinger.OnFinish = func(stats *ping.Statistics) {
		t.Logf("\n--- %s ping statistics ---\n", stats.Addr)
		t.Logf("%d packets transmitted, %d packets received, %v%% packet loss\n",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
		t.Logf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
			stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
	}

	pinger.Count = 5
	pinger.Timeout = 10 * time.Second
	pinger.Source = "10.60.0.1"

	time.Sleep(3 * time.Second)

	pinger.Run()

	time.Sleep(1 * time.Second)

	stats := pinger.Statistics()
	if stats.PacketsSent != stats.PacketsRecv {
		t.Fatal("Ping Failed")
		return
	}
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