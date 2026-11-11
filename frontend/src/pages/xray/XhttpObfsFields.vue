<script setup lang="ts">
import { XField, XFieldGrid, XInput, XNumber, XSelect, XSwitch, type Option } from '../../kit'
import { sessionIdLength, sessionIdWrong } from './xhttp'

/**
 * What an XHTTP client and its server must agree on beyond the path: the
 * padding and how it is hidden, the uplink's HTTP method, where the session
 * id, the sequence and (packet-up) the uploaded data go. The inbound window
 * has it for its server and the clients its links set up, the outbound
 * window for a client of another such server. The slot "method" puts a field
 * beside the uplink method, e.g. the server's header limit.
 */
const props = defineProps<{ ns: Record<string, any> }>()

/** A table comes with a length long enough for it, as Xray needs one; a length that is not stays marked. */
function setTable(table: string) {
  props.ns.sessionIDTable = table
  if (table && sessionIdWrong(props.ns)) props.ns.sessionIDLength = sessionIdLength(table)
}

const list = (values: string[]): Option[] => values.map((v) => ({ value: v, label: v }))
const placements = (dflt: string, values: string[]): Option[] => [{ value: '', label: `Default (${dflt})` }, ...list(values)]
const paddingPlacements = placements('queryInHeader', ['queryInHeader', 'header', 'cookie', 'query'])
const paddingMethods = placements('repeat-x', ['repeat-x', 'tokenish'])
const uplinkMethods: Option[] = [
  { value: '', label: 'Default (POST)' },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'GET', label: 'GET (packet-up only)' },
]
const idPlacements = placements('path', ['path', 'header', 'cookie', 'query'])
const dataPlacements = placements('body', ['body', 'header', 'cookie', 'query'])
const idTables: Option[] = [{ value: '', label: 'Default (UUID)' }, ...list(['number', 'hex', 'HEX', 'base36', 'BASE36', 'Base62', 'alphabet'])]
</script>

<template>
  <XFieldGrid>
    <XField v-slot="a" label="Padding Bytes"><XInput v-bind="a" v-model.trim="ns.xPaddingBytes" mono /></XField>
  </XFieldGrid>
  <XSwitch v-model="ns.xPaddingObfsMode" boxed label="Padding Obfs Mode" />
  <XFieldGrid v-if="ns.xPaddingObfsMode">
    <XField v-slot="a" label="Padding Key"><XInput v-bind="a" v-model.trim="ns.xPaddingKey" mono placeholder="x_padding" /></XField>
    <XField v-slot="a" label="Padding Header"><XInput v-bind="a" v-model.trim="ns.xPaddingHeader" mono placeholder="X-Padding" /></XField>
    <XField v-slot="a" label="Padding Placement"><XSelect v-bind="a" :model-value="ns.xPaddingPlacement ?? ''" @update:model-value="ns.xPaddingPlacement = String($event ?? '')" :options="paddingPlacements" mono /></XField>
    <XField v-slot="a" label="Padding Method"><XSelect v-bind="a" :model-value="ns.xPaddingMethod ?? ''" @update:model-value="ns.xPaddingMethod = String($event ?? '')" :options="paddingMethods" mono /></XField>
  </XFieldGrid>
  <XFieldGrid>
    <XField v-slot="a" label="Uplink HTTP Method"><XSelect v-bind="a" :model-value="ns.uplinkHTTPMethod ?? ''" @update:model-value="ns.uplinkHTTPMethod = String($event ?? '')" :options="uplinkMethods" mono /></XField>
    <slot name="method" />
  </XFieldGrid>
  <XFieldGrid>
    <XField v-slot="a" label="Session ID Placement"><XSelect v-bind="a" :model-value="ns.sessionIDPlacement ?? ''" @update:model-value="ns.sessionIDPlacement = String($event ?? '')" :options="idPlacements" mono /></XField>
    <XField v-if="ns.sessionIDPlacement && ns.sessionIDPlacement !== 'path'" v-slot="a" label="Session ID Key">
      <XInput v-bind="a" v-model.trim="ns.sessionIDKey" mono placeholder="x_session" />
    </XField>
    <XField v-slot="a" label="Session ID Table"><XSelect v-bind="a" :model-value="ns.sessionIDTable ?? ''" @update:model-value="setTable(String($event ?? ''))" :options="idTables" mono /></XField>
    <XField v-if="ns.sessionIDTable" v-slot="a" label="Session ID Length"><XInput v-bind="a" v-model.trim="ns.sessionIDLength" mono :placeholder="sessionIdLength(ns.sessionIDTable)" :invalid="sessionIdWrong(ns)" /></XField>
    <XField v-slot="a" label="Sequence Placement"><XSelect v-bind="a" :model-value="ns.seqPlacement ?? ''" @update:model-value="ns.seqPlacement = String($event ?? '')" :options="idPlacements" mono /></XField>
    <XField v-if="ns.seqPlacement && ns.seqPlacement !== 'path'" v-slot="a" label="Sequence Key">
      <XInput v-bind="a" v-model.trim="ns.seqKey" mono placeholder="x_seq" />
    </XField>
  </XFieldGrid>
  <XFieldGrid v-if="ns.mode === 'packet-up'" :cols="3">
    <XField v-slot="a" label="Uplink Data Placement"><XSelect v-bind="a" :model-value="ns.uplinkDataPlacement ?? ''" @update:model-value="ns.uplinkDataPlacement = String($event ?? '')" :options="dataPlacements" mono /></XField>
    <template v-if="ns.uplinkDataPlacement && ns.uplinkDataPlacement !== 'body'">
      <XField v-slot="a" label="Uplink Data Key"><XInput v-bind="a" v-model.trim="ns.uplinkDataKey" mono placeholder="x_data" /></XField>
      <XField v-slot="a" label="Uplink Chunk Size"><XNumber v-bind="a" v-model="ns.uplinkChunkSize" block :min="0" placeholder="0 (unlimited)" /></XField>
    </template>
  </XFieldGrid>
</template>
