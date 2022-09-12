using System;
using System.Threading.Tasks;
using Data;
using UnityEngine.Networking;

namespace Network
{
    [Serializable]
    public class ListPresentResponse : CommonResponse
    {
        public UserPresent[] presents;
        public bool isNext;
    }
    
    [Serializable]
    public class ListPresentApi : ApiBase<CommonRequest, ListPresentResponse>
    {
        public override string Path => $"/user/{GameManager.userData.user.id}/present/index/{_index}";

        public override string Method => UnityWebRequest.kHttpVerbGET;

        private int _index;

        public ListPresentApi(int index)
        {
            _index = index;
        }
    }

    public partial class ApiClient
    {
        public async Task<ListPresentResponse> ListPresentAsync(int index)
        {
            var api = new ListPresentApi(index);
            var res = await Post(api);
            return res;
        }
    }
}
