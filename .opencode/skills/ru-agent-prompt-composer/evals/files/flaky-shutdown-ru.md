Есть странный flake maybe на shutdown или drain.
То ли `context canceled` где-то теряется, то ли worker не останавливается и потом тест иногда висит.
Не хочу просто увеличить timeout.
Надо сначала понять, где именно ломается, скорее bootstrap или health/readiness штука, и потом уже чинить аккуратно.
Посмотри еще race или integration angle тоже.
