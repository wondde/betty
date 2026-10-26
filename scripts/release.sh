#!/usr/bin/env bash
# Builds, signs (Developer ID), notarizes and publishes the Mac app so teammates get it from
# /download and running apps offer a one-click update.
#   scripts/release.sh "이번 버전에서 바뀐 점 한 줄"
# Needs: the Developer ID Application certificate and the betty-notary notarytool profile
# (xcrun notarytool store-credentials betty-notary …). Bump the version in betty.go,
# build/config.yml and build/darwin/Info.plist first.
set -euo pipefail
cd "$(dirname "$0")/.."
NOTES=${1:-}
IDENTITY=${BETTY_IDENTITY:-"Developer ID Application: SeungWon Shin (4JAB44S28T)"}
HOST=${BETTY_RELEASE_HOST:?BETTY_RELEASE_HOST를 정해 주세요}
DEST=${BETTY_RELEASE_DIR:?BETTY_RELEASE_DIR을 정해 주세요}
V=$(grep -o 'const version = "[^"]*"' betty.go | cut -d'"' -f2)
OUT=bin/dist

echo "1/6 빌드 ($V, Apple 칩 + 인텔)"
~/go/bin/wails3 task darwin:package:universal >/dev/null
ARCHS=$(lipo -archs bin/betty.app/Contents/MacOS/betty)
echo "   $ARCHS"
[[ "$ARCHS" == *x86_64* && "$ARCHS" == *arm64* ]] || { echo "   인텔/Apple 칩 둘 다 들어 있지 않아요"; exit 1; }
[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' bin/betty.app/Contents/Info.plist)" = "$V" ] || { echo "   Info.plist 버전이 $V 와 달라요"; exit 1; }

echo "2/6 Developer ID 서명"
rm -rf "$OUT" && mkdir -p "$OUT/stage"
cp -R bin/betty.app "$OUT/stage/Betty.app"
codesign --force --options runtime --timestamp --sign "$IDENTITY" "$OUT/stage/Betty.app"
codesign --verify --deep --strict "$OUT/stage/Betty.app"
ln -s /Applications "$OUT/stage/Applications"

echo "3/6 DMG 만들기"
hdiutil create -volname Betty -srcfolder "$OUT/stage" -ov -format UDZO "$OUT/Betty-$V.dmg" >/dev/null
codesign --force --timestamp --sign "$IDENTITY" "$OUT/Betty-$V.dmg"

echo "4/6 Apple 공증 (몇 분 걸려요)"
xcrun notarytool submit "$OUT/Betty-$V.dmg" --keychain-profile betty-notary --wait | grep -E "status:|id:" | tail -2
xcrun stapler staple "$OUT/Betty-$V.dmg" >/dev/null
xcrun stapler staple "$OUT/stage/Betty.app" >/dev/null # the DMG's ticket covers the app inside
spctl -a -t open --context context:primary-signature "$OUT/Betty-$V.dmg"
spctl -a -t exec "$OUT/stage/Betty.app"

echo "5/6 업데이트용 zip, latest.json"
ditto -c -k --keepParent "$OUT/stage/Betty.app" "$OUT/Betty-$V.zip"
SHA=$(shasum -a 256 "$OUT/Betty-$V.zip" | cut -c1-64)
SIZE=$(stat -f %z "$OUT/Betty-$V.dmg")
node -e 'const [v,n,s,z,a]=process.argv.slice(1);console.log(JSON.stringify({version:v,notes:n,dmg:`Betty-${v}.dmg`,zip:`Betty-${v}.zip`,sha256:s,size:Number(z),archs:a.split(" "),published_at:new Date().toISOString()},null,1))' "$V" "$NOTES" "$SHA" "$SIZE" "$ARCHS" > "$OUT/latest.json"

echo "6/6 서버에 올리기 (latest.json은 마지막에)"
ssh -n "$HOST" "mkdir -p $DEST"
scp -q "$OUT/Betty-$V.dmg" "$OUT/Betty-$V.zip" "$HOST:$DEST/"
scp -q "$OUT/latest.json" "$HOST:$DEST/latest.json.tmp"
ssh -n "$HOST" "cd $DEST && [ \"\$(shasum -a 256 Betty-$V.zip | cut -c1-64)\" = $SHA ] && mv latest.json.tmp latest.json && ls -la"
echo "   https://ticket.blueberry-team.com/download"
