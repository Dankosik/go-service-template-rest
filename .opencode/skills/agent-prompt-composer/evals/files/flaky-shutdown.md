hay un flake raro, maybe around shutdown / drain, not sure.
Sometimes `context canceled` gets swallowed or the worker does not stop and then the test hangs.
No quiero simplemente subir el timeout. Don't just increase timeouts.
First figure out where it breaks, probably bootstrap or the health/readiness thing, then fix it carefully.
Also check the race or integration angle if that is the right proof path.
