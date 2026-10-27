package com.json2proxy.shared

import kotlinx.coroutines.CoroutineStart
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

class ProxyControllerTest {
    @Test
    fun connectThenDisconnect() = runTest {
        val controller = ProxyController()
        val notice = async(start = CoroutineStart.UNDISPATCHED) { controller.notices.first() }

        controller.connect("example.com:8443")
        assertEquals(ConnectionState.Connected("example.com:8443"), controller.state.value)
        assertEquals(Notice.Connected("example.com:8443"), notice.await())

        controller.disconnect()
        assertEquals(ConnectionState.Disconnected, controller.state.value)
    }

    @Test
    fun blankServerFails() = runTest {
        val controller = ProxyController()
        val notice = async(start = CoroutineStart.UNDISPATCHED) { controller.notices.first() }

        controller.connect(" ")
        assertIs<ConnectionState.Failed>(controller.state.value)
        assertEquals(NoticeLevel.Error, notice.await().level)
    }
}
