using System;
using System.Threading.Tasks;
using Data;
using UnityEngine.Networking;

namespace Network
{
    [Serializable]
    public class ListItemResponse : CommonResponse
    {
        [Serializable]
        public class GachaData
        {
            public GachaMaster gacha;
            public GachaItemMaster[] gachaItemList;
        }
        
        public string oneTimeToken;
        public User user;
        public UserItem[] items;
        public UserCard[] cards;
    }
    
    [Serializable]
    public class ListItemApi : ApiBase<CommonRequest, ListItemResponse>
    {
        public override string Path { get; } = $"/user/{GameManager.userData.user.id}/item";
        public override string Method => UnityWebRequest.kHttpVerbGET;
    }

    public partial class ApiClient
    {
        public async Task<ListItemResponse> ListItemAsync()
        {
            var api = new ListItemApi();
            var res = await Post(api);
            this._oneTimeToken = res.oneTimeToken;
            return res;
        }
    }
}
