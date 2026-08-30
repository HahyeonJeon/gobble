// Package stop owns evidence that Run and Resume stop through context
// cancellation. Cancellation is structured, state remains inspectable, and
// occupancy remains active until Release. Recovery is Inspect, Release, then
// Resume; this owner adds no Cancel verb or process signaling contract. It does
// not own an assay graph or fixture.
package stop
