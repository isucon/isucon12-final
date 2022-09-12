using System;
using System.Threading.Tasks;
using Data;
using UnityEngine.Networking;

namespace Network
{
    [Serializable]
    public class ListGachaResponse : CommonResponse
    {
        [Serializable]
        public class GachaData
        {
            public GachaMaster gacha;
            public GachaItemMaster[] gachaItemList;
        }
        
        public string oneTimeToken;
        public GachaData[] gachas;
    }
    
    [Serializable]
    public class ListGachaApi : ApiBase<CommonRequest, ListGachaResponse>
    {
        public override string Path { get; } = $"/user/{GameManager.userData.user.id}/gacha/index";
        public override string Method => UnityWebRequest.kHttpVerbGET;
    }

    public partial class ApiClient
    {
        public async Task<ListGachaResponse> ListGachaAsync()
        {
            var api = new ListGachaApi();
            var res = await Post(api);
            this._oneTimeToken = res.oneTimeToken;
            return res;
        }
    }
}
