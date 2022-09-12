using System;
using System.Threading.Tasks;
using Data;

namespace Network
{
    [Serializable]
    public class DrawGachaRequest : CommonRequest
    {
        public string oneTimeToken;
    }

    [Serializable]
    public class DrawGachaResponse : CommonResponse
    {
        public UserPresent[] presents;
    }
    
    public class DrawGachaApi : ApiBase<DrawGachaRequest, DrawGachaResponse>
    {
        public override string Path => $"/user/{GameManager.userData.user.id}/gacha/draw/{_gachaId}/{_gachaCount}";

        private long _gachaId;
        private int _gachaCount;

        public DrawGachaApi(long gachaId, int gachaCount, DrawGachaRequest req)
        {
            _gachaId = gachaId;
            _gachaCount = gachaCount;

            RequestData = req;
        }
    }

    public partial class ApiClient
    {
        public async Task<DrawGachaResponse> DrawGachaAsync(long gachaId, int gachaCount)
        {
            var req = new DrawGachaRequest()
            {
                oneTimeToken = _oneTimeToken,
            };
            var api = new DrawGachaApi(gachaId, gachaCount, req);
            var res = await Post(api);
            return res;
        }
    }
}
