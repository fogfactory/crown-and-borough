# Changelog

## [0.10.1](https://github.com/fogfactory/crown-and-borough/compare/v0.10.0...v0.10.1) (2026-09-23)


### Bug Fixes

* **dev:** copy web/.env.local into Claude Code worktrees ([bb986a8](https://github.com/fogfactory/crown-and-borough/commit/bb986a88879d891ea7a0e0d69c7b4fdd7eb6fdb1))
* **dev:** copy web/.env.local into Claude Code worktrees ([fe23eb4](https://github.com/fogfactory/crown-and-borough/commit/fe23eb4cd4652119295a322abc943e8561d06b57))
* **rules:** align player rules with engine and balance ([62f43bf](https://github.com/fogfactory/crown-and-borough/commit/62f43bf17008dee514719f0c228bbc7fdf012e76))
* **rules:** align player rules with engine and balance ([8715d27](https://github.com/fogfactory/crown-and-borough/commit/8715d27026f5080dbc9c7c33f4371b397b8561a2))
* **store:** skip players without an emitting noble when waiting for submissions ([861ce9e](https://github.com/fogfactory/crown-and-borough/commit/861ce9e271e67bcb7d7e3258008dab6f1ecca384))
* **store:** skip players without an emitting noble when waiting for submissions ([3614f6c](https://github.com/fogfactory/crown-and-borough/commit/3614f6c66c7e232f32efa4023e02de86c6c28a60))

## [0.10.0](https://github.com/fogfactory/crown-and-borough/compare/v0.9.1...v0.10.0) (2026-09-23)


### Features

* **web:** winter orders overlay with validation and diagnostics ([d09d500](https://github.com/fogfactory/crown-and-borough/commit/d09d50066010dfbf6bf433c097cd1fe7054df876))
* **web:** winter orders overlay with validation and diagnostics ([12cf1b7](https://github.com/fogfactory/crown-and-borough/commit/12cf1b71f79a0a451fbb01ec30f6649a935e2ace))


### Refactoring

* **web:** share hotseat and online game screen internals ([915bf35](https://github.com/fogfactory/crown-and-borough/commit/915bf35d77900b6e414b026826abf5a703842778))

## [0.9.1](https://github.com/fogfactory/crown-and-borough/compare/v0.9.0...v0.9.1) (2026-09-18)


### Bug Fixes

* **map:** scatter and shrink winter snowflakes ([d46f839](https://github.com/fogfactory/crown-and-borough/commit/d46f8391fc82e315d3b9d8c6469d5307943fe76e))
* **map:** scatter and shrink winter snowflakes ([0f9b1d0](https://github.com/fogfactory/crown-and-borough/commit/0f9b1d0f9e75189063656b859b44dca77ba2028d))

## [0.9.0](https://github.com/fogfactory/crown-and-borough/compare/v0.8.0...v0.9.0) (2026-09-17)


### Features

* **map:** add touch gestures, pinch zoom and on-screen map controls ([2a1012a](https://github.com/fogfactory/crown-and-borough/commit/2a1012a26a55bd5084ae50e090cf59a4bfc500cf))
* **map:** game-icons markers and bolder terrain patterns ([a1e9406](https://github.com/fogfactory/crown-and-borough/commit/a1e9406687e334ef4b62fe2982bab526a1bc7da4))
* **map:** ownership markers, dark frontier casing and terrain textures ([99999b0](https://github.com/fogfactory/crown-and-borough/commit/99999b06fe9ee3b8349bb3330a42789a54e75666))
* **map:** pastel terrain textures, larger markers and winter snow ([a653fec](https://github.com/fogfactory/crown-and-borough/commit/a653fecf4d42525ff0b63c6f1945cac9dd6a968a))
* **web:** add per-player order submission status dots ([38954b4](https://github.com/fogfactory/crown-and-borough/commit/38954b4cb7d34d75bd95ca8c70ac4a2073e7df2c))
* **web:** banner brand mark for headers and favicon ([5735c30](https://github.com/fogfactory/crown-and-borough/commit/5735c30041c420794d21243b3b25b0af8931a5a7))
* **web:** smartphone-ready game shell, compact header and density pass ([d7ffc52](https://github.com/fogfactory/crown-and-borough/commit/d7ffc5202998cfb2ec76fd64374ca5639ba48fb2))
* **web:** smartphone-ready UX overhaul ([e7dac12](https://github.com/fogfactory/crown-and-borough/commit/e7dac12e52b7460f2f37cec7f81c129dcd1c0dd3))


### Bug Fixes

* **engine:** keep player colors away from terrain hues ([ee5a1e4](https://github.com/fogfactory/crown-and-borough/commit/ee5a1e44a733501f0aaa9504246b7d91123f201d))
* **map:** preserve map aspect ratio on any container ([0ddafc0](https://github.com/fogfactory/crown-and-borough/commit/0ddafc02f2df0fc712eb00827744918668d0cbbf))
* **web:** fit the game shell in the viewport, demote scores/lobby ([d8745c9](https://github.com/fogfactory/crown-and-borough/commit/d8745c930f04a5360d58842ea5096243ab92474f))

## [0.8.0](https://github.com/fogfactory/crown-and-borough/compare/v0.7.0...v0.8.0) (2026-09-14)


### Features

* **engine:** retreat priority buckets, friendly merge and castle self-capture ([dd98a78](https://github.com/fogfactory/crown-and-borough/commit/dd98a78c29addc7a156bca61b62c1f6583bf6bd0))


### Bug Fixes

* **engine:** allow friendly join and disperse fusion ([be5a92c](https://github.com/fogfactory/crown-and-borough/commit/be5a92cc155d608208d7ba21c467597a723aa750))
* **engine:** let joins follow allied attack winners ([5de2585](https://github.com/fogfactory/crown-and-borough/commit/5de25852728b28843316001c26065c3ec5f5992d))
* **engine:** prioritize retreats and support friendly castle capture ([030efaf](https://github.com/fogfactory/crown-and-borough/commit/030efaf99b37b0069b324a308a25833bf74809de))
* **engine:** stack friendly disperse arrivals ([f667b5f](https://github.com/fogfactory/crown-and-borough/commit/f667b5f12f391b480a146399ca10c2339540726f))
* **retreat:** remove unreachable army ID tie-break ([9dbf8a6](https://github.com/fogfactory/crown-and-borough/commit/9dbf8a6860dbdafca7564d9c3dc5bf6fc3b0862e))


### Documentation

* **retreat:** fix English tie-break paragraph ([23a3a1d](https://github.com/fogfactory/crown-and-borough/commit/23a3a1d2a67bb10e729c80fa7645629b45dfcf56))
* **retreat:** use ascending territory trigram instead of army id in tie-break ([71583ff](https://github.com/fogfactory/crown-and-borough/commit/71583ffa4155db910ae578956ea133d9158e2aac))

## [0.7.0](https://github.com/fogfactory/crown-and-borough/compare/v0.6.0...v0.7.0) (2026-09-10)


### Features

* **orders:** estimate winter order costs ([fc75ad6](https://github.com/fogfactory/crown-and-borough/commit/fc75ad638f44dd1af9b2ab00e90ad5d29e5e06ed))
* **orders:** estimate winter order costs ([483210a](https://github.com/fogfactory/crown-and-borough/commit/483210ac0e1ed62198ec8688fb5d790bd5d371b7))
* **orders:** surface winter order syntax errors ([b028eca](https://github.com/fogfactory/crown-and-borough/commit/b028ecafe15bccd6e359cce0d7bfddceafa90294))

## [0.6.0](https://github.com/fogfactory/crown-and-borough/compare/v0.5.0...v0.6.0) (2026-09-09)


### Features

* **online:** réhydrater ma dernière soumission + note si diverge ([feff620](https://github.com/fogfactory/crown-and-borough/commit/feff620d6215fad88df1d56e5dacbea9f8b0214c))
* **online:** réhydrater ma dernière soumission + note si diverge ([0c4109f](https://github.com/fogfactory/crown-and-borough/commit/0c4109fc82512e13ce623855a3591bf4133bacae))

## [0.5.0](https://github.com/fogfactory/crown-and-borough/compare/v0.4.4...v0.5.0) (2026-09-08)


### Features

* **balance:** rebalance terrain rations ([e39c928](https://github.com/fogfactory/crown-and-borough/commit/e39c928ed6c7109d6ce500671475ef98b99c8656))
* **balance:** rebalance terrain rations ([6fec885](https://github.com/fogfactory/crown-and-borough/commit/6fec885d0c0e3f0130ebc774873d91f827bc1a62))
* **economy:** cap mill level at 3 with progressive costs ([9242c4e](https://github.com/fogfactory/crown-and-borough/commit/9242c4e5e10644ac97d91100d964eab4675fec0c))
* **economy:** cap mill level at 3 with progressive costs ([36bb7f4](https://github.com/fogfactory/crown-and-borough/commit/36bb7f47be7354249ef89c1f7efa7bb5415eb01f))
* **engine:** add resource transfer orders ([1213e30](https://github.com/fogfactory/crown-and-borough/commit/1213e30ac078572b03d8e3a51a33f48fce85d434))
* **engine:** add resource transfer orders ([1bc7214](https://github.com/fogfactory/crown-and-borough/commit/1bc7214245ddae489baa258352cbc805ab87c11b))
* **mapgen:** double terrain seed density ([f51cc7c](https://github.com/fogfactory/crown-and-borough/commit/f51cc7cd69b471ee8f6402049e4eda7843865e60))
* **mapgen:** double terrain seed density (sites/8 -&gt; sites/4) ([c1d26ea](https://github.com/fogfactory/crown-and-borough/commit/c1d26ea138d35436a2a79453a12c259152f9b325))
* **mapgen:** enforce two vertex-disjoint paths ([0c01ff7](https://github.com/fogfactory/crown-and-borough/commit/0c01ff7507d8dd97fbb7127808559a3a221086d4))
* **mapgen:** enforce two vertex-disjoint paths ([3fdef12](https://github.com/fogfactory/crown-and-borough/commit/3fdef1249c3f568848fc2e4a9987e0f50da78d93))
* **release:** promote develop to main for 0.5.0 ([02122eb](https://github.com/fogfactory/crown-and-borough/commit/02122eb0956eb53f79746358c61a2a17fc977ee3))


### Bug Fixes

* **docs:** template mill costs from balance ([bb7199c](https://github.com/fogfactory/crown-and-borough/commit/bb7199c81c2cd25bfcb16b190a48554639b11a13))
* **engine:** block supply only on enemy armies ([7885f61](https://github.com/fogfactory/crown-and-borough/commit/7885f612c8c4e661babb4bea8e39441fa5c56805))
* **engine:** block supply only on enemy armies ([70c791c](https://github.com/fogfactory/crown-and-borough/commit/70c791cbb383454c99cbb397884e1075c3e80fdd))
* **engine:** consume local terrain production ([beae901](https://github.com/fogfactory/crown-and-borough/commit/beae9019d0aea4968b862354561acbc4f61a0c96))
* **ui:** preview action transfer routes ([9312ae0](https://github.com/fogfactory/crown-and-borough/commit/9312ae0034b3c4ac2b74b8b6811e68cd3039e3b2))
* **ui:** preview action transfer routes ([7e98281](https://github.com/fogfactory/crown-and-borough/commit/7e98281ea4b16895fab68804c63bb9bd30ae8e6e))

## [0.4.4](https://github.com/fogfactory/crown-and-borough/compare/v0.4.3...v0.4.4) (2026-09-05)


### Bug Fixes

* **ui:** mark pending loop orders with ? in overlay ([fba67a3](https://github.com/fogfactory/crown-and-borough/commit/fba67a3e5480f6f45136c59e2c78ed8e232ee1aa))
* **ui:** mark pending loop orders with ? in overlay ([e87bf24](https://github.com/fogfactory/crown-and-borough/commit/e87bf24c8e55b23d94dfe2fbb489da0304d85f02))

## [0.4.3](https://github.com/fogfactory/crown-and-borough/compare/v0.4.2...v0.4.3) (2026-09-04)


### Bug Fixes

* **engine:** reject non-adjacent order chains ([00b60db](https://github.com/fogfactory/crown-and-borough/commit/00b60db22fa9b4536e866e2d60f0518ff8b85424))
* **map:** distinguish draft intentions visually ([b9b5b55](https://github.com/fogfactory/crown-and-borough/commit/b9b5b55a02c68b19aba1a7c92034f2d7d53df1ec))
* **map:** overlay local des intentions (brouillons + chaînes connues) ([d1986fc](https://github.com/fogfactory/crown-and-borough/commit/d1986fc6589cdd2d968f2487f43ecf5a3caa526a))
* **map:** overlay local des intentions (brouillons + chaînes connues) ([fcd896b](https://github.com/fogfactory/crown-and-borough/commit/fcd896b4ee8d4300759d0ffb41ced14993af78f5))
* **map:** render complete order chains with outlined arrows ([e91affb](https://github.com/fogfactory/crown-and-borough/commit/e91affba4189933bfce6a1deade555f43fd89f41))
* **map:** show first draft order in overlay ([8b9e59d](https://github.com/fogfactory/crown-and-borough/commit/8b9e59ddb203dd4cf67c0358e167e4e8d3015895))
* **ui:** expose rules and FAQ pages ([6285083](https://github.com/fogfactory/crown-and-borough/commit/6285083b80afc5d5f998808b9299a723244a2593))
* **ui:** expose rules and FAQ pages ([722326a](https://github.com/fogfactory/crown-and-borough/commit/722326a2125e7910acd2f34349ed6eb8a789f54c))


### Documentation

* **rules:** corrige la distribution des rations locales — une armée par case ([891c1f8](https://github.com/fogfactory/crown-and-borough/commit/891c1f821b95d03a1ccf037a671f3d988ddb086e))
* **rules:** lot règles unique — famine, moulins, dispersion, synchro join, FAQ ([8312b64](https://github.com/fogfactory/crown-and-borough/commit/8312b6400b1fdf2d6ae3d5b6dcf3f8dccfc8b0f5))
* **rules:** lot règles unique — famine, moulins, dispersion, synchro join, FAQ ([2fc4cc2](https://github.com/fogfactory/crown-and-borough/commit/2fc4cc23d585c9640aaff618548f9a357ddbb81c))

## [0.4.2](https://github.com/fogfactory/crown-and-borough/compare/v0.4.1...v0.4.2) (2026-09-01)


### Bug Fixes

* **online:** converge army and report details with hotseat ([af14e85](https://github.com/fogfactory/crown-and-borough/commit/af14e85c621bd5d318300eca28a683b454a2169d))
* **online:** converge army details with hotseat ([90a9dfa](https://github.com/fogfactory/crown-and-borough/commit/90a9dfa9d4e2257aeb118b1a42d9581e5da91009))


### Refactoring

* **web:** share report pane between hotseat and online ([d783a90](https://github.com/fogfactory/crown-and-borough/commit/d783a90de0779893491a41bb21d87c7336cdb91f))

## [0.4.1](https://github.com/fogfactory/crown-and-borough/compare/v0.4.0...v0.4.1) (2026-08-26)


### Bug Fixes

* **deploy:** avoid curl smoke test pipe errors ([7667ae8](https://github.com/fogfactory/crown-and-borough/commit/7667ae8a2d3f3aa663c47ca5d0a7776f6b33ea2b))
* **deploy:** avoid curl smoke test pipe errors ([4312989](https://github.com/fogfactory/crown-and-borough/commit/431298916dcff003e89d01e190ae3d72227dfc5e))

## [0.4.0](https://github.com/fogfactory/crown-and-borough/compare/v0.3.1...v0.4.0) (2026-08-26)


### Features

* **deploy:** display deployed version ([c92d356](https://github.com/fogfactory/crown-and-borough/commit/c92d356ac8689f1c2e1b7e56d58d2eac0268d942))
* **deploy:** display deployed version ([bb1ef7e](https://github.com/fogfactory/crown-and-borough/commit/bb1ef7eb6df1ff920547982b60fc1d8a4aa08132))
* **workflow:** define main and develop release flow ([518b7dd](https://github.com/fogfactory/crown-and-borough/commit/518b7dde8bc160a674e2d957d19f7b6ab0de902c))


### Bug Fixes

* **workflow:** skip empty branch synchronization ([639b8a6](https://github.com/fogfactory/crown-and-borough/commit/639b8a6fcb38bca5a81faae1f5e76972aa1380e0))


### Documentation

* **workflow:** document promotion merge strategy ([7996e25](https://github.com/fogfactory/crown-and-borough/commit/7996e250e5d59a428cec0cbc67752f20dcfaa61e))
