package host

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/internal/common"
	"github.com/openshift/assisted-service/internal/events"
	"github.com/openshift/assisted-service/internal/hardware"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("instructionmanager", func() {
	var (
		ctx               = context.Background()
		host              models.Host
		db                *gorm.DB
		mockEvents        *events.MockHandler
		stepsReply        models.Steps
		hostId, clusterId strfmt.UUID
		stepsErr          error
		instMng           *InstructionManager
		ctrl              *gomock.Controller
		hwValidator       *hardware.MockValidator
		instructionConfig InstructionConfig
		dbName            = "instructionmanager"
	)

	BeforeEach(func() {
		db = common.PrepareTestDB(dbName)
		ctrl = gomock.NewController(GinkgoT())
		mockEvents = events.NewMockHandler(ctrl)
		hwValidator = hardware.NewMockValidator(ctrl)
		instMng = NewInstructionManager(getTestLog(), db, hwValidator, instructionConfig, nil)
		hostId = strfmt.UUID(uuid.New().String())
		clusterId = strfmt.UUID(uuid.New().String())
		host = getTestHost(hostId, clusterId, "unknown invalid state")
		host.Role = models.HostRoleMaster
		host.Inventory = masterInventory()
		Expect(db.Create(&host).Error).ShouldNot(HaveOccurred())
	})
	Context("No DHCP", func() {
		BeforeEach(func() {
			cluster := common.Cluster{Cluster: models.Cluster{ID: &clusterId, VipDhcpAllocation: swag.Bool(false), MachineNetworkCidr: "1.2.3.0/24"}}
			Expect(db.Create(&cluster).Error).ShouldNot(HaveOccurred())
		})
		Context("get_next_steps", func() {
			It("invalid_host_state", func() {
				stepsReply, stepsErr = instMng.GetNextSteps(ctx, &host)
				Expect(stepsReply.Instructions).To(HaveLen(0))
				Expect(stepsErr).Should(BeNil())
			})
			It("discovering", func() {
				checkStepsByState(HostStatusDiscovering, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInventory, models.StepTypeConnectivityCheck})
			})
			It("known", func() {
				checkStepsByState(HostStatusKnown, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeConnectivityCheck, models.StepTypeFreeNetworkAddresses})
			})
			It("disconnected", func() {
				checkStepsByState(HostStatusDisconnected, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInventory, models.StepTypeConnectivityCheck})
			})
			It("insufficient", func() {
				checkStepsByState(HostStatusInsufficient, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInventory, models.StepTypeConnectivityCheck, models.StepTypeFreeNetworkAddresses})
			})
			It("pending-for-input", func() {
				checkStepsByState(HostStatusPendingForInput, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInventory, models.StepTypeConnectivityCheck, models.StepTypeFreeNetworkAddresses})
			})
			It("error", func() {
				checkStepsByState(HostStatusError, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeExecute})
			})
			It("installing", func() {
				checkStepsByState(HostStatusInstalling, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInstall})
			})
			It("reset", func() {
				checkStepsByState(HostStatusResetting, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeResetInstallation})
			})
		})
	})

	Context("With DHCP", func() {
		BeforeEach(func() {
			cluster := common.Cluster{Cluster: models.Cluster{ID: &clusterId, VipDhcpAllocation: swag.Bool(true), MachineNetworkCidr: "1.2.3.0/24"}}
			Expect(db.Create(&cluster).Error).ShouldNot(HaveOccurred())
		})
		Context("get_next_steps", func() {
			It("invalid_host_state", func() {
				stepsReply, stepsErr = instMng.GetNextSteps(ctx, &host)
				Expect(stepsReply.Instructions).To(HaveLen(0))
				Expect(stepsErr).Should(BeNil())
			})
			It("discovering", func() {
				checkStepsByState(HostStatusDiscovering, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInventory, models.StepTypeConnectivityCheck})
			})
			It("known", func() {
				checkStepsByState(HostStatusKnown, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeConnectivityCheck, models.StepTypeFreeNetworkAddresses, models.StepTypeDhcpLeaseAllocate})
			})
			It("disconnected", func() {
				checkStepsByState(HostStatusDisconnected, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInventory, models.StepTypeConnectivityCheck})
			})
			It("insufficient", func() {
				checkStepsByState(HostStatusInsufficient, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInventory, models.StepTypeConnectivityCheck, models.StepTypeFreeNetworkAddresses, models.StepTypeDhcpLeaseAllocate})
			})
			It("pending-for-input", func() {
				checkStepsByState(HostStatusPendingForInput, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInventory, models.StepTypeConnectivityCheck, models.StepTypeFreeNetworkAddresses, models.StepTypeDhcpLeaseAllocate})
			})
			It("error", func() {
				checkStepsByState(HostStatusError, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeExecute})
			})
			It("installing", func() {
				checkStepsByState(HostStatusInstalling, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeInstall, models.StepTypeDhcpLeaseAllocate})
			})
			It("installing-in-progress", func() {
				checkStepsByState(models.HostStatusInstallingInProgress, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeDhcpLeaseAllocate})
			})
			It("reset", func() {
				checkStepsByState(HostStatusResetting, &host, db, mockEvents, instMng, hwValidator, ctx,
					[]models.StepType{models.StepTypeResetInstallation})
			})
		})
	})

	AfterEach(func() {
		// cleanup
		common.DeleteTestDB(db, dbName)
		ctrl.Finish()
		stepsReply = models.Steps{}
		stepsErr = nil
	})

})

func checkStepsByState(state string, host *models.Host, db *gorm.DB, mockEvents *events.MockHandler, instMng *InstructionManager, mockValidator *hardware.MockValidator, ctx context.Context,
	expectedStepTypes []models.StepType) {
	mockEvents.EXPECT().AddEvent(gomock.Any(), host.ID.String(), common.GetEventSeverityFromHostStatus(state), gomock.Any(), gomock.Any(), host.ClusterID.String())
	updateReply, updateErr := updateHostStatus(ctx, getTestLog(), db, mockEvents, host.ClusterID, *host.ID, *host.Status, state, "")
	ExpectWithOffset(1, updateErr).ShouldNot(HaveOccurred())
	ExpectWithOffset(1, updateReply).ShouldNot(BeNil())
	h := getHost(*host.ID, host.ClusterID, db)
	ExpectWithOffset(1, swag.StringValue(h.Status)).Should(Equal(state))
	validDiskSize := int64(128849018880)
	var disks = []*models.Disk{
		{DriveType: "disk", Name: "sdb", SizeBytes: validDiskSize},
		{DriveType: "disk", Name: "sda", SizeBytes: validDiskSize},
		{DriveType: "disk", Name: "sdh", SizeBytes: validDiskSize},
	}
	mockValidator.EXPECT().GetHostValidDisks(gomock.Any()).Return(disks, nil).AnyTimes()
	stepsReply, stepsErr := instMng.GetNextSteps(ctx, h)
	ExpectWithOffset(1, stepsReply.Instructions).To(HaveLen(len(expectedStepTypes)))
	if stateValues, ok := instMng.stateToSteps[state]; ok {
		Expect(stepsReply.NextInstructionSeconds).Should(Equal(stateValues.NextStepInSec))
	} else {
		Expect(stepsReply.NextInstructionSeconds).Should(Equal(defaultNextInstructionInSec))
	}

	for i, step := range stepsReply.Instructions {
		ExpectWithOffset(1, step.StepType).Should(Equal(expectedStepTypes[i]))
	}
	ExpectWithOffset(1, stepsErr).ShouldNot(HaveOccurred())
}
