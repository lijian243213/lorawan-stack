// Copyright © 2020 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import React from 'react'
import { defineMessages } from 'react-intl'

import Form from '@ttn-lw/components/form'
import Select from '@ttn-lw/components/select'
import Checkbox from '@ttn-lw/components/checkbox'
import Input from '@ttn-lw/components/input'
import KeyValueMap from '@ttn-lw/components/key-value-map'

import Message from '@ttn-lw/lib/components/message'

import PropTypes from '@ttn-lw/lib/prop-types'

import { ACTIVATION_MODES, DEVICE_CLASSES } from '@console/lib/device-utils'

const m = defineMessages({
  delayValue: '{count, plural, one {{count} second} other {{count} seconds}}',
  pingSlotPeriodicityDescription: 'Periodicity of the class B ping slot',
  pingSlotPeriodicityTitle: 'Ping Slot Periodicity',
  pingSlotPeriodicityValue: '{count, plural, one {every second} other {every {count} seconds}}',
  pingSlotFrequencyTitle: 'Ping Slot Frequency',
  pingSlotFrequencyDescription: 'Frequency of the class B ping slot (Hz)',
  resetsFCnt: 'Resets Frame Counters',
  resetWarning: 'Resetting is insecure and makes your device susceptible for replay attacks',
  rx1DelayTitle: 'RX1 Delay',
  rx1DelayDescription: 'Class A RX1 delay in seconds. RX2 delay is RX1 delay + 1 second.',
  rx1DataRateOffsetTitle: 'RX1 Data Rate Offset',
  rx2DataDateIndexDescription:
    'The default RX2 data rate index value the device uses after a reset',
  rx2DataRateIndexTitle: 'RX2 Data Rate Index',
  rx2FrequencyTitle: 'RX2 Frequency',
  rx2FrequencyDescription: 'Frequency for RX2 (Hz)',
  frequencyPlaceholder: 'e.g. 869525000 for 869,525 MHz',
  macSettings: 'MAC settings',
  updateSuccess: 'The MAC settings have been updated successfully',
  factoryPresetFreqTitle: 'Factory Preset Frequencies',
  factoryPresetFreqDescription: 'List of factory-preset frequencies. Note: order is respected.',
  freqAdd: 'Add Frequency',
})

// Inconsistent mac_settings enum values see https://github.com/TheThingsNetwork/lorawan-stack/issues/2258

// 0...15
const dataRateIndexOptions = Array.from({ length: 16 }, (_, index) => ({
  value: index,
  label: index,
}))

// 1...15
const rx1DelayOptions = Array.from({ length: 15 }, (_, index) => ({
  value: index + 1,
  label: <Message content={m.delayValue} values={{ count: index + 1 }} />,
}))

// 0...7
const pingSlotPeriodicityOptions = Array.from({ length: 8 }, (_, index) => {
  const value = Math.pow(2, index)

  return {
    value: `PING_EVERY_${value}S`,
    label: <Message content={m.pingSlotPeriodicityValue} values={{ count: value }} />,
  }
})

// 0...7
const dataRateOffsetOptions = Array.from({ length: 8 }, (_, index) => ({
  value: index,
  label: index,
}))

const MacSettingsSection = props => {
  const { activationMode, deviceClasses, resetsFCnt: initialFCnt, initiallyCollapsed } = props

  const isClassB = deviceClasses.includes(DEVICE_CLASSES.CLASS_B)
  const isMulticast = activationMode === ACTIVATION_MODES.MULTICAST
  const isOTAA = activationMode === ACTIVATION_MODES.OTAA
  const isABP = activationMode === ACTIVATION_MODES.ABP

  const [resetsFCnt, setResetsFCnt] = React.useState(isABP && initialFCnt)
  const handleResetsFCntChange = React.useCallback(evt => {
    const { checked } = evt.target

    setResetsFCnt(checked)
  }, [])

  return (
    <Form.CollapseSection
      id="mac-settings"
      title={m.macSettings}
      initiallyCollapsed={initiallyCollapsed}
    >
      {!isMulticast && (
        <>
          {isABP && (
            <Form.Field
              title={m.rx1DelayTitle}
              description={m.rx1DelayDescription}
              name="mac_settings.rx1_delay.value"
              component={Select}
              options={rx1DelayOptions}
            />
          )}
          {(isABP || isOTAA) && (
            <Form.Field
              title={m.rx1DataRateOffsetTitle}
              name="mac_settings.rx1_data_rate_offset"
              component={Select}
              options={dataRateOffsetOptions}
            />
          )}
          {isABP && (
            <Form.Field
              title={m.resetsFCnt}
              onChange={handleResetsFCntChange}
              warning={resetsFCnt ? m.resetWarning : undefined}
              name="mac_settings.resets_f_cnt"
              component={Checkbox}
            />
          )}
        </>
      )}
      <Form.Field
        title={m.rx2DataRateIndexTitle}
        description={m.rx2DataDateIndexDescription}
        name="mac_settings.rx2_data_rate_index.value"
        component={Select}
        options={dataRateIndexOptions}
      />
      <Form.Field
        type="number"
        min={100000}
        step={100}
        title={m.rx2FrequencyTitle}
        description={m.rx2FrequencyDescription}
        placeholder={m.frequencyPlaceholder}
        name="mac_settings.rx2_frequency"
        component={Input}
      />
      <Form.Field
        indexAsKey
        name="mac_settings.factory_preset_frequencies"
        component={KeyValueMap}
        title={m.factoryPresetFreqTitle}
        description={m.factoryPresetFreqDescription}
        addMessage={m.freqAdd}
        valuePlaceholder={m.frequencyPlaceholder}
      />
      {isClassB && (
        <>
          <Form.Field
            title={m.pingSlotPeriodicityTitle}
            description={m.pingSlotPeriodicityDescription}
            name="mac_settings.ping_slot_periodicity.value"
            component={Select}
            options={pingSlotPeriodicityOptions}
          />
          <Form.Field
            type="number"
            min={100000}
            step={100}
            title={m.pingSlotFrequencyTitle}
            description={m.pingSlotFrequencyDescription}
            placeholder={m.frequencyPlaceholder}
            name="mac_settings.ping_slot_frequency"
            component={Input}
          />
        </>
      )}
    </Form.CollapseSection>
  )
}

MacSettingsSection.propTypes = {
  activationMode: PropTypes.oneOf(Object.values(ACTIVATION_MODES)).isRequired,
  deviceClasses: PropTypes.arrayOf(PropTypes.oneOf(Object.values(DEVICE_CLASSES))).isRequired,
  initiallyCollapsed: PropTypes.bool,
  resetsFCnt: PropTypes.bool,
}

MacSettingsSection.defaultProps = {
  resetsFCnt: false,
  initiallyCollapsed: true,
}

export default MacSettingsSection
