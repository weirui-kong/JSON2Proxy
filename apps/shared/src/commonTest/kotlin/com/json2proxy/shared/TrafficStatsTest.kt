package com.json2proxy.shared

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class TrafficStatsTest {
    @Test
    fun speedsComeFromSuccessiveSamples() {
        val sampler = TrafficSampler()
        val first = sampler.sample("""{"time":1000,"tcp":2,"udp":1,"up":100,"down":1000}""")!!
        assertEquals(0, first.uploadSpeed)
        assertEquals(0, first.downloadSpeed)
        assertEquals(2, first.tcpConnections)
        assertEquals(1, first.udpConnections)

        val second = sampler.sample("""{"time":3000,"tcp":1,"udp":0,"up":2100,"down":9000}""")!!
        assertEquals(1000, second.uploadSpeed)
        assertEquals(4000, second.downloadSpeed)
        assertEquals(2100, second.uploadTotal)

        // Counters went down: the engine restarted.
        val restarted = sampler.sample("""{"time":4000,"up":10,"down":10}""")!!
        assertEquals(0, restarted.uploadSpeed)
        assertEquals(0, restarted.downloadSpeed)
    }

    @Test
    fun hostSpeedsAndDetail() {
        val sampler = TrafficSampler()
        sampler.sample(
            """{"time":0,"up":0,"down":0,"hosts":[{"host":"a.test","up":0,"down":500,"tcp":1,"action":"PROXY","rule":"FINAL,PROXY","last":0}]}""",
        )
        val stats = sampler.sample(
            """{"time":500,"up":0,"down":0,
               "hosts":[{"host":"a.test","up":50,"down":1500,"tcp":1,"action":"PROXY","rule":"FINAL,PROXY","last":400},
                        {"host":"b.test","up":10,"down":10,"udp":1,"action":"DIRECT","rule":"GEOIP,CN,DIRECT","last":450}],
               "dns":[{"name":"b.test","addrs":["192.0.2.1"],"error":"","time":450}],
               "flows":[{"time":450,"network":"udp","host":"b.test","port":53,"action":"DIRECT","rule":"GEOIP,CN,DIRECT","error":""}]}""",
        )!!
        val a = stats.hosts.first { it.host == "a.test" }
        assertEquals(100, a.uploadSpeed)
        assertEquals(2000, a.downloadSpeed)
        val b = stats.hosts.first { it.host == "b.test" }
        assertEquals(0, b.downloadSpeed) // first time seen
        assertEquals("DIRECT", b.action)
        assertEquals(listOf("192.0.2.1"), stats.dns.single().addrs)
        assertEquals(53, stats.flows.single().port)
    }

    @Test
    fun emptyOrBrokenSnapshots() {
        val sampler = TrafficSampler()
        assertNull(sampler.sample(""))
        assertNull(sampler.sample("{not json"))
    }

    @Test
    fun formatsBytesWithKMGOnly() {
        assertEquals("0.00 KB/s", TrafficFormat.rate(0, TrafficUnit.Bytes))
        assertEquals("0.29 KB/s", TrafficFormat.rate(300, TrafficUnit.Bytes))
        assertEquals("1.00 KB/s", TrafficFormat.rate(1024, TrafficUnit.Bytes))
        assertEquals("12.5 KB/s", TrafficFormat.rate(12_800, TrafficUnit.Bytes))
        assertEquals("500 KB/s", TrafficFormat.rate(512_000, TrafficUnit.Bytes))
        assertEquals("1023 KB/s", TrafficFormat.rate(1_048_000, TrafficUnit.Bytes))
        assertEquals("1.00 MB/s", TrafficFormat.rate(1_048_100, TrafficUnit.Bytes))
        assertEquals("2.50 GB/s", TrafficFormat.rate(2_684_354_560, TrafficUnit.Bytes))
        assertEquals("2048 GB/s", TrafficFormat.rate(2_199_023_255_552, TrafficUnit.Bytes))
    }

    @Test
    fun formatsBitsInThousands() {
        assertEquals("0.80 Kb/s", TrafficFormat.rate(100, TrafficUnit.Bits))
        assertEquals("8.00 Mb/s", TrafficFormat.rate(1_000_000, TrafficUnit.Bits))
        assertEquals("80.0 Mb/s", TrafficFormat.rate(10_000_000, TrafficUnit.Bits))
        assertEquals("1.00 Gb/s", TrafficFormat.rate(125_000_000, TrafficUnit.Bits))
    }
}
